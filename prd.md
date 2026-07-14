# PRD: `async` — JavaScript-style Concurrency for Go

**Status:** Draft v0.1
**Author:** Raunak
**Last updated:** 14 July 2026

---

## 1. Overview

Go's concurrency primitives (goroutines, channels, `sync.WaitGroup`, `select`) are powerful but low-level. Developers arriving from JavaScript — one of the largest migration paths into Go — already have a well-formed mental model for concurrency built on Promises, `async/await`, and combinators like `Promise.all`. Today they must abandon that model entirely and relearn concurrency from first principles, which slows onboarding and produces buggy code (leaked goroutines, unhandled panics, missing cancellation).

`async` is a Go library that maps the JavaScript concurrency mental model onto idiomatic Go, using generics for type safety, `context.Context` for cancellation, and `(T, error)` returns instead of rejection chains.

**Positioning:** an onboarding ramp and boilerplate-killer for the 90% case — not a replacement for channels.

## 2. Problem statement

A JS developer writing their first concurrent Go code must simultaneously learn: goroutine lifecycle management, channel semantics (buffered vs unbuffered, closing rules), `WaitGroup` bookkeeping, `select` statements, context propagation, and panic recovery. The equivalent JS operation is often one line (`await Promise.all([...])`). This gap produces three recurring failure modes in production Go written by ex-JS developers:

1. **Goroutine leaks** — fire-and-forget goroutines with no lifecycle owner.
2. **Process crashes** — an unrecovered panic in any goroutine kills the whole binary (in JS, an unhandled rejection is survivable).
3. **Missing cancellation** — sibling work continues after one task fails, wasting resources.

## 3. Goals

- Provide a type-safe, JS-shaped API for the most common concurrency patterns.
- Make cancellation and panic safety the default, not opt-in.
- Keep error handling idiomatic Go: `(T, error)`, `errors.Is/As` compatible.
- Zero dependencies beyond the standard library.

## 4. Non-goals

- Replacing channels or `select` for complex coordination (fan-in routing, pipelines).
- Reimplementing the JS event loop or microtask queue semantics.
- Streams/observables (possible v2 territory).
- Actor frameworks or distributed concurrency.

## 5. Target users

- **Primary:** JS/TS developers in their first 0–12 months of Go.
- **Secondary:** experienced Go developers who want less `WaitGroup`/`errgroup` boilerplate for common patterns.

## 6. Concept mapping: JavaScript → Go wrapper

This is the core of the product. Each JS concept, what Go currently requires, and what `async` provides.

| # | JS concept | Raw Go equivalent | `async` wrapper |
|---|-----------|-------------------|-----------------|
| 1 | `new Promise(executor)` / calling an async function | `go func(){}()` + result channel | `async.Go[T](ctx, fn) *Promise[T]` — derives a cancellable child context for the task |
| 2 | `await p` | `<-ch` with error channel or struct | `p.Await(ctx) (T, error)` — this ctx bounds the *wait*, not the task |
| 3 | `Promise.all([...])` | `errgroup.Group` + manual result slice + mutex/index bookkeeping | `async.All(ctx, ps...) ([]T, error)` — cancels siblings on first error |
| 4 | `Promise.allSettled([...])` | `WaitGroup` + result struct slice | `async.AllSettled(ctx, ps...) []Result[T]` — never fails |
| 5 | `Promise.race([...])` | `select` over N channels (needs reflection or codegen for dynamic N) | `async.Race(ctx, ps...) (T, error)` — first to settle wins |
| 6 | `Promise.any([...])` | Custom: first success wins, aggregate errors | `async.Any(ctx, ps...) (T, error)` — returns `AggregateError` if all fail |
| 7 | `.then(fn)` | Manual chaining via new goroutine | `async.Then(p, fn) *Promise[U]` (package-level func: Go methods can't add type params) |
| 8 | `.catch(fn)` | `if err != nil` | Not wrapped — `Await` returns `error`; use normal Go error handling. Deliberate design choice. |
| 9 | `.finally(fn)` | `defer` | `p.Finally(fn)` (thin sugar; docs point to `defer`) |
| 10 | `try/catch` around `await` | `if err != nil` + `recover()` in every goroutine | Built-in: every `async.Go` recovers panics and surfaces them as `*PanicError` |
| 11 | Unhandled rejection ≠ crash | Unrecovered panic = process death | Panic containment by default (see #10) |
| 12 | `AbortController` / `signal` | `context.Context` + `ctx.Done()` checks | `ctx` threaded through every API; `p.Cancel()` sugar over `context.CancelFunc` |
| 13 | `AbortSignal.timeout(ms)` | `context.WithTimeout` + cleanup | `async.WithTimeout(p, d)` — `Await` returns `ErrTimeout` (wraps `context.DeadlineExceeded`) |
| 14 | `setTimeout(fn, ms)` | `time.AfterFunc` / `time.Sleep` in goroutine | `async.Delay(ctx, d, fn)` — cancellable via ctx |
| 15 | `setInterval(fn, ms)` | `time.Ticker` + goroutine + stop bookkeeping | `async.Interval(ctx, d, fn) (stop func())` |
| 16 | `p-limit` / `p-map` (bounded concurrency over a collection) | Semaphore channel + `WaitGroup` + indexed results | `async.Map(ctx, items, fn, async.WithLimit(n)) ([]U, error)` |
| 17 | `p-queue` (worker pool) | Hand-rolled pool: jobs channel, N workers, results channel | `async.Pool[T](n)` with `Submit(fn) *Promise[T]` and `Drain(ctx)` |
| 18 | `p-retry` (retry with backoff) | Loop + backoff math + jitter + ctx checks | `async.Retry(ctx, fn, async.Attempts(3), async.Backoff(...))` |
| 19 | Debounce / throttle (lodash) | Timer + mutex state machine | `async.Debounce(fn, d)`, `async.Throttle(fn, d)` |
| 20 | `queueMicrotask` / event-loop ordering | No equivalent (Go scheduler is preemptive) | Not wrapped — documented in the migration guide as a mental-model difference |
| 21 | `EventEmitter` | Channels + fan-out bookkeeping | Out of scope for v1 (candidate for v2 as `async.Emitter[T]`) |

## 7. API design principles

**Errors, not rejections.** `Await` returns `(T, error)`. There is no `.Catch`. This keeps orchestration JS-shaped but error handling Go-shaped, and keeps the library compatible with `errors.Is`, `errors.As`, and wrapped errors.

**Context everywhere — with two distinct roles.** Every spawning API (`Go`, `Map`, `Retry`, `Delay`, `Interval`) takes `context.Context` as its first argument; this is the *task context*, bound at spawn time. `Go` derives a cancellable child from it, so every `Promise` carries its own cancel handle. `Await(ctx)` takes a separate *wait context* that bounds only how long the caller blocks — cancelling it abandons the wait without killing the task. Combinators exploit the spawn-time handle: when `All` sees one failure, it cancels the sibling tasks via their handles. Spawn-time binding mirrors JS more closely than it first appears (`fetch(url, { signal })` binds the abort signal at call time), and sibling cancellation is a capability JS largely lacks — the migration guide sells it as an upgrade, not a tax.

**Panic safety by default.** `async.Go` wraps every function in `recover()`. A panic becomes a `*PanicError` carrying the panic value and stack trace, returned from `Await` like any other error. The process never dies because of a task the library owns.

**No goroutine leaks.** Every `Promise` has an owner and a completion path. Abandoned promises (never awaited) are safe: the goroutine completes, stores its result, and exits. `Interval`, `Pool`, and `Debounce` all return explicit stop/drain mechanisms tied to context.

**Generics, no reflection.** All type safety is compile-time. No `interface{}` in the public API.

## 8. Illustrative usage

```go
// JS: const [user, orders] = await Promise.all([getUser(id), getOrders(id)])
userP := async.Go(ctx, func(ctx context.Context) (User, error) { return getUser(ctx, id) })
ordersP := async.Go(ctx, func(ctx context.Context) ([]Order, error) { return getOrders(ctx, id) })

user, err := userP.Await(ctx)
orders, err2 := ordersP.Await(ctx)

// Heterogeneous join sugar:
user, orders, err := async.All2(ctx, userP, ordersP)

// JS: await pMap(ids, fetchUser, { concurrency: 10 })
users, err := async.Map(ctx, ids, fetchUser, async.WithLimit(10))

// JS: await pRetry(flaky, { retries: 3 })
res, err := async.Retry(ctx, flaky, async.Attempts(3), async.ExpBackoff(100*time.Millisecond))
```

Note: `All` over a homogeneous slice returns `[]T`; heterogeneous joins use generated `All2`…`All8` variants (Go generics cannot express variadic heterogeneous tuples).

## 9. Deliverables

**v0.1 (core):** `Go`, `Await`, `All`/`All2`–`All8`, `Race`, `AllSettled`, panic containment, context propagation, `WithTimeout`.

**v0.2 (collections):** `Map`, `ForEach`, `WithLimit`, `Pool`.

**v0.3 (utilities):** `Retry`, `Delay`, `Interval`, `Debounce`, `Throttle`, `Any` + `AggregateError`.

**Docs:** a "coming from JavaScript" migration guide is a first-class deliverable — every JS pattern above with side-by-side raw-Go and `async` versions, plus a section on genuine mental-model differences (no event loop, preemptive scheduling, no microtasks).

## 10. Risks and mitigations

- **"Not idiomatic Go" pushback.** Mitigate through positioning (onboarding ramp, 90% case), a zero-magic implementation readable in an afternoon, and never hiding `context`.
- **Overlap with `conc`/`errgroup`.** Differentiate on the coherent JS-shaped surface and the migration guide; interoperate rather than compete (accept/return standard types only).
- **Generics limitations** (no method-level type params, no heterogeneous variadics). Mitigate with package-level functions (`async.Then`) and small code-generated `AllN` variants.
- **API sprawl.** Guard with the non-goals list; anything channel-shaped stays out.

## 11. Success metrics

- Time-to-first-correct-concurrent-PR for a JS-background hire (qualitative, via user interviews).
- GitHub adoption: stars, imports (pkg.go.dev importers count), issues opened by non-authors.
- Zero goroutine-leak and zero unrecovered-panic reports in library-owned code paths.