# Coming from JavaScript

You already think in Promises, `async`/`await`, and `Promise.all`. This guide
maps that model onto `async`, shows the raw-Go boilerplate it replaces, and —
just as importantly — calls out the places where Go is **not** JavaScript, so the
ramp doesn't quietly become a trap.

- [The 60-second map](#the-60-second-map)
- [Side by side](#side-by-side)
- [Mental-model differences you must internalize](#mental-model-differences-you-must-internalize)
- [When to put the library down and use channels](#when-to-put-the-library-down-and-use-channels)

## The 60-second map

| JavaScript | `async` |
|---|---|
| `new Promise(fn)` / calling an `async` function | `promise.New(fn)` |
| `Promise.resolve(v)` / `Promise.reject(e)` | `promise.Resolve(v)` / `promise.Reject(e)` |
| `await p` | `promise.Await(p)` or `p.Await()` |
| `p.then(fn)` | `promise.Then(p, fn)` |
| `p.finally(fn)` | `p.Finally(fn)` |
| `p.catch(fn)` | — use the `error` from `Await` (see [Errors](#2-errors-not-rejections)) |
| `Promise.all([...])` | `promise.All(...)` / `promise.All2`–`All8` for mixed types |
| `Promise.race([...])` | `promise.Race(...)` |
| `Promise.allSettled([...])` | `promise.AllSettled(...)` |
| `Promise.any([...])` | `promise.Any(...)` → `AggregateError` if all fail |
| `pMap(items, fn, { concurrency })` | `collections.Map(items, fn, collections.Concurrency(n))` |
| `new PQueue({ concurrency })` | `collections.NewQueue(n)` (`Add`, `OnIdle`) |
| `pRetry(fn, { retries })` | `promise.Retry(fn, promise.Attempts(n))` |
| `AbortSignal.timeout(ms)` | `promise.Timeout(p, d)` |
| `new AbortController()` / `signal` | `abort.NewController()` / `abort.Signal` |
| `setTimeout` / `setInterval` | `timers.SetTimeout` / `timers.SetInterval` |
| `_.debounce` / `_.throttle` | `timers.Debounce` / `timers.Throttle` |

## Side by side

Each pattern below is shown three ways: the **JavaScript** you'd write, the
**raw Go** you'd otherwise have to write, and the **`async`** version.

### Run two things concurrently and join

```js
// JavaScript
const [user, orders] = await Promise.all([getUser(id), getOrders(id)]);
```

```go
// Raw Go: errgroup + shared vars, or channels + a struct
var user User
var orders []Order
g, ctx := errgroup.WithContext(ctx)
g.Go(func() error { var e error; user, e = getUser(ctx, id); return e })
g.Go(func() error { var e error; orders, e = getOrders(ctx, id); return e })
if err := g.Wait(); err != nil { /* ... */ }
```

```go
// async — heterogeneous join keeps each type
user, orders, err := promise.All2(getUser(id), getOrders(id))
```

### Bounded concurrency over a list

```js
// JavaScript
const users = await pMap(ids, getUser, { concurrency: 10 });
```

```go
// Raw Go: semaphore channel + WaitGroup + indexed results
results := make([]User, len(ids))
sem := make(chan struct{}, 10)
g, ctx := errgroup.WithContext(ctx)
for i, id := range ids {
	i, id := i, id
	g.Go(func() error {
		sem <- struct{}{}; defer func() { <-sem }()
		u, err := getUser(ctx, id)
		results[i] = u
		return err
	})
}
err := g.Wait()
```

```go
// async
users, err := collections.Map(ids, getUser, collections.Concurrency(10))
```

### Retry with backoff

```js
// JavaScript
const res = await pRetry(flaky, { retries: 2 });
```

```go
// Raw Go: loop + backoff math + context checks
var res T
var err error
delay := 100 * time.Millisecond
for attempt := 1; attempt <= 3; attempt++ {
	if res, err = flaky(ctx); err == nil { break }
	if attempt < 3 {
		select {
		case <-time.After(delay):
		case <-ctx.Done(): return ctx.Err()
		}
		delay *= 2
	}
}
```

```go
// async
res, err := promise.Await(promise.Retry(flaky, promise.Attempts(3), promise.ExpBackoff(100*time.Millisecond)))
```

### Cancel in flight

```js
// JavaScript
const c = new AbortController();
const p = fetchThing({ signal: c.signal });
c.abort();
```

```go
// async — WithSignal opts the work into cancellation, like { signal }
c := abort.NewController()
p := promise.WithSignal(func(signal *abort.Signal) (Thing, error) {
	return fetchThing(signal.Context()) // signal.Context() bridges to any ctx-aware API
})
c.Abort()
```

## Mental-model differences you must internalize

The orchestration reads like JavaScript. The runtime does not. These are the
things that will bite you if you assume otherwise.

### 1. No event loop, no microtask queue

JavaScript runs your code on a single thread and interleaves work through the
event loop; `await` yields to the microtask queue in a well-defined order. Go has
**none of that**. Goroutines run on multiple OS threads, truly in parallel, and
the scheduler can switch between them at almost any point. There is no
`queueMicrotask`, and no ordering guarantee between two concurrent tasks unless
you create one with a channel or a lock.

### 2. Errors, not rejections

There is no `try/catch` around `await`, and no `.catch`. Every `Await` returns
`(value, error)`:

```go
v, err := promise.Await(p)
if err != nil {
	// handle it — errors.Is / errors.As work through the chain
}
```

A panic (Go's version of a thrown exception) is **contained**, not propagated: it
comes back as a `*promise.PanicError`, so one task crashing never takes down the
process — the survivable-rejection guarantee you're used to, made explicit.

### 3. True parallelism means you need real synchronization

Because goroutines run in parallel, shared mutable state is a data race. In JS a
single-threaded model let you mutate freely between `await`s; in Go you cannot.
If two tasks touch the same variable, guard it with `sync.Mutex` or `sync/atomic`.
This library does not — and cannot — hide that.

### 4. Cancellation is cooperative and opt-in

Neither Go nor JavaScript can forcibly stop a running function. In JS you thread
a `signal` into `fetch`; in `async` you use `promise.WithSignal` and watch the
signal. Plain `promise.New` work **cannot be interrupted mid-flight** — `Abort`
marks it aborted for awaiters, but the function runs to completion. Design
long-running tasks to accept a signal.

### 5. `.then` chains nest instead of flowing

Go methods can't add type parameters, so `Then` is a package-level function.
Cross-type chains read inside-out:

```go
// JS:  getUser(id).then(u => u.Name).then(greet)
promise.Then(promise.Then(getUser(id), userName), greet)
```

In practice, prefer straight-line Go — `Await`, then act on the value — over deep
`Then` chains. The chaining exists for familiarity, not because it's the idiom.

### 6. Every promise is a goroutine you own

There is no "floating promise" warning, but there is also no free lunch: a
promise you never await still runs its function to completion and then exits. It
won't leak, but it won't stop on its own either. If you start work you might not
need, start it with `WithSignal` and `Abort` it.

## When to put the library down and use channels

`async` is built for one shape: **fan out a known set of tasks, then gather the
results** (with optional concurrency limits, retries, timeouts, and
cancellation). That covers a large fraction of everyday concurrency. It is *not*
a general toolkit, and reaching past it is expected, not a failure.

Drop to raw goroutines, channels, `select`, and `sync` when you need:

- **Streaming / pipelines** — producer→consumer, or results consumed as they
  arrive rather than all at once.
- **Fan-in / dynamic routing** — merging many sources, `select` over several
  channels, non-blocking checks.
- **Long-lived coordination** — background workers, tickers driving state,
  request-scoped `context.Context` threaded deep through a service.
- **Shared-state synchronization** — `Mutex`, `RWMutex`, `atomic`, `Once`,
  `Cond`, `singleflight`.

The library returns and accepts only standard types, so you can mix the two
freely: use `async` for the 90% that's fan-out/gather, and reach for channels the
moment the shape changes.
