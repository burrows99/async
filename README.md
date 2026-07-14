# async

![async — JavaScript-style concurrency for Go](docs/banner.png)

[![CI](https://github.com/burrows99/async/actions/workflows/ci.yml/badge.svg)](https://github.com/burrows99/async/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/burrows99/async.svg)](https://pkg.go.dev/github.com/burrows99/async)
[![Go Report Card](https://goreportcard.com/badge/github.com/burrows99/async)](https://goreportcard.com/report/github.com/burrows99/async)
[![standard-readme compliant](https://img.shields.io/badge/readme%20style-standard-brightgreen.svg?style=flat-square)](https://github.com/RichardLitt/standard-readme)

JavaScript-style concurrency for Go — Promises, async/await, and combinators over goroutines and context.

`async` brings the concurrency vocabulary you already know from JavaScript —
`Promise`, `await`, `Promise.all`, `AbortController` — to Go, so orchestration
code reads like the JS it mirrors while goroutines and `context.Context` run
underneath.

```go
// JS: const [user, orders, recs] = await Promise.all([getUser(1), getOrders(1), getRecs(1)]);
user, orders, recs, err := promise.All3(getUser(1), getOrders(1), getRecs(1))
```

It is an onboarding ramp and a boilerplate-killer for the 90% case — not a
replacement for channels. When you outgrow it, drop back to goroutines and
`select`; `async` only accepts and returns standard types, so it interoperates
rather than competes. It has zero dependencies beyond the standard library.

## Table of Contents

- [Background](#background)
- [Install](#install)
- [Usage](#usage)
- [Packages](#packages)
- [Scope](#scope)
- [Cancellation and errors](#cancellation-and-errors)
- [Roadmap](#roadmap)
- [API](#api)
- [Maintainers](#maintainers)
- [Contributing](#contributing)
- [License](#license)

## Background

JavaScript is one of the largest migration paths into Go, and developers arrive
with a well-formed mental model for concurrency: Promises, `async`/`await`, and
combinators like `Promise.all`. In Go they must relearn concurrency from
goroutines, channels, `sync.WaitGroup`, and `select` — which slows onboarding
and produces buggy code (leaked goroutines, unrecovered panics, missing
cancellation).

`async` maps that JavaScript model onto idiomatic Go: generics for type safety,
`(T, error)` returns instead of rejection chains, panic containment by default,
and cancellation modelled on `AbortController`/`AbortSignal`. The public API uses
the JavaScript and npm vocabulary throughout (`Promise.all`, `p-map`'s
`{ concurrency }`, `p-queue`'s `add`/`onIdle`, lodash `debounce`/`throttle`), so
the surface is familiar while the semantics stay Go-shaped.

New to Go from JavaScript? Start with **[Coming from JavaScript](docs/coming-from-javascript.md)** —
side-by-side JS, raw Go, and `async`, plus the mental-model differences that will
bite you if you assume Go is just JavaScript with types.

## Install

```sh
go get github.com/burrows99/async
```

Requires Go 1.23 or newer. There are no dependencies beyond the Go standard
library. See the reference docs on [pkg.go.dev](https://pkg.go.dev/github.com/burrows99/async).

## Usage

Put the `promise.New` wrapping in your function definitions — exactly where JS
puts the `async` keyword — and return a `*promise.Promise[T]`. Then every call
site reads like the JavaScript it mirrors:

```go
import (
	"github.com/burrows99/async/promise"
	"github.com/burrows99/async/collections"
)

// This is our "async function". Calling it returns a Promise, just like
// `async function getUser(id) { ... }` does in JS.
func getUser(id int) *promise.Promise[User] {
	return promise.New(func() (User, error) {
		return db.FindUser(id)
	})
}

func example(ids []int) {
	// JS: const user = await getUser(1);
	user, err := promise.Await(getUser(1))

	// JS: const users = await Promise.all([getUser(1), getUser(2), getUser(3)]);
	users, err := promise.All(getUser(1), getUser(2), getUser(3))

	// JS: const [user, orders] = await Promise.all([getUser(1), getOrders(1)]);
	user, orders, err := promise.All2(getUser(1), getOrders(1))

	// JS: const users = await pMap(ids, getUser, { concurrency: 10 });
	users, err = collections.Map(ids, getUser, collections.Concurrency(10))
}
```

For a full, runnable showcase of every pattern — written to read like JavaScript
line for line — see [`examples/dashboard`](examples/dashboard):

```sh
go run ./examples/dashboard
```

## Packages

The library is organized by JavaScript concept, so each import reads like the API
it provides:

| Package | Mirrors | Provides |
|---------|---------|----------|
| [`promise`](promise) | `Promise` | `New`, `Resolve`, `Reject`, `WithSignal`, `Await`, `Then`, `Finally`, `All`, `All2`–`All8`, `Race`, `AllSettled`, `Any` + `AggregateError`, `Retry`, `Timeout` |
| [`abort`](abort) | `AbortController` / `AbortSignal` | `NewController`, `Controller`, `Signal` |
| [`collections`](collections) | `p-map` / `p-queue` | `Map`, `ForEach`, `Concurrency`, `Queue` |
| [`timers`](timers) | `setTimeout` / lodash | `SetTimeout`, `SetInterval`, `Debounce`, `Throttle` |

## Scope

`async` is deliberately a *slice* of concurrency, not the whole thing. It does one
job well — **fan out a known set of tasks, then gather the results**, with
optional concurrency limits, retries, timeouts, and cancellation (`All`, `Any`,
`Race`, `Map`, `Queue`, `Retry`). For that shape it's a real
errgroup/semaphore boilerplate-killer, and it names that job honestly rather than
claiming to be a full toolkit.

Reach for raw goroutines, channels, `select`, and `sync` when you need:

- **Shared-state synchronization** — `Mutex`, `atomic`, `Once`, `sync.Map`. A
  promise library doesn't replace a mutex, and real code needs them constantly.
- **Streaming, pipelines, and fan-in** — producer→consumer, or results consumed
  as they arrive. `Map` returns everything at once, in order — it doesn't stream.
- **General `select`** — waiting on many dynamic events, non-blocking checks, or
  ticker-driven coordination.
- **Deep `context.Context` propagation** — `async` hides context by design
  (which is what makes it approachable), so a service that threads request
  context end-to-end for deadlines and tracing should use `context` directly.
  `signal.Context()` bridges at a boundary, but it is not full propagation.

Everything here returns and accepts only standard types, so you can mix the two
freely: use `async` for the fan-out/gather 90%, and drop to channels the moment
the shape changes. Coming from JavaScript? The
[migration guide](docs/coming-from-javascript.md) spells out exactly where the
ramp ends.

## Cancellation and errors

Go cannot preempt a goroutine, and neither can JavaScript preempt a running
function. That is why JS cancellation is opt-in: you thread an `AbortSignal` into
`fetch(url, { signal })`, and work that ignores the signal keeps running.
`async` works the same way.

- **Plain work** uses `promise.New(func() (T, error))` — no signal. It cannot be
  interrupted mid-flight; `Abort` marks it aborted for awaiters, but the function
  runs to completion.
- **Cancellable work** uses `promise.WithSignal(func(signal *abort.Signal) (T, error))`
  and watches the signal — the Go analogue of accepting `{ signal }`.

```go
// JS: const c = new AbortController(); doWork(c.signal); c.abort();
c := abort.NewController()
p := promise.New(func() (T, error) { /* … watch c.Signal().Done() … */ })
c.Abort()
```

`signal.Context()` bridges to the standard library — pass it to any
context-aware Go API (database, HTTP, gRPC) so the underlying call is cancelled
too. It is the one place a `context.Context` surfaces, exactly as a signal
surfaces only at the `fetch` call in JavaScript.

Errors stay Go-shaped. `Await` returns `(T, error)`; there is no `.Catch`, and
errors compose with `errors.Is`/`errors.As`. A panic never crashes the process:
every function runs under `recover()`, and a panic becomes a
`*promise.PanicError` carrying the value and a stack trace, returned from `Await`
like any other error.

```go
p := promise.New(func() (int, error) { panic("boom") })
_, err := promise.Await(p)

var pe *promise.PanicError
if errors.As(err, &pe) {
	log.Printf("recovered: %v\n%s", pe.Value, pe.Stack)
}
```

## Roadmap

- **Core (v0.1):** `promise` (New, WithSignal, Await, All/All2–All8, Race,
  AllSettled, Timeout, panic containment) + `abort` (Controller, Signal).
  ✅ implemented
- **Collections (v0.2):** `collections` — `Map`, `ForEach`, `Concurrency`,
  `Queue` (bounded concurrency over a slice, à la p-map / p-queue). ✅ implemented
- **Utilities (v0.3):** `promise.Any` + `AggregateError`, `promise.Retry`, and
  `timers` — `SetTimeout`, `SetInterval`, `Debounce`, `Throttle`. ✅ implemented
- **Docs:** [Coming from JavaScript](docs/coming-from-javascript.md) — every
  pattern side by side with raw Go, plus the mental-model differences.
  ✅ implemented

## API

Full, generated API documentation lives on
[pkg.go.dev/github.com/burrows99/async](https://pkg.go.dev/github.com/burrows99/async),
one page per package:

- [`promise`](https://pkg.go.dev/github.com/burrows99/async/promise) — promises and combinators
- [`abort`](https://pkg.go.dev/github.com/burrows99/async/abort) — `AbortController` / `AbortSignal`
- [`collections`](https://pkg.go.dev/github.com/burrows99/async/collections) — `Map`, `ForEach`, `Queue`
- [`timers`](https://pkg.go.dev/github.com/burrows99/async/timers) — `SetTimeout`, `Debounce`, `Throttle`

## Maintainers

[@burrows99](https://github.com/burrows99) (Raunak Burrows).

## Contributing

Issues and pull requests are welcome. Ask questions or propose changes via
[GitHub Issues](https://github.com/burrows99/async/issues) or
[Discussions](https://github.com/burrows99/async/discussions). PRs are accepted;
small, focused changes are easiest to review, and no CLA or commit sign-off is
required.

The `All2`…`All8` variants in `promise/alln.go` are generated — Go generics have
no variadic type parameters. Edit `internal/gen`, then regenerate:

```sh
go generate ./...
```

Before opening a PR, make sure the same checks CI runs pass locally:

```sh
gofmt -l .
go vet ./...
go test -race ./...
```

## License

[MIT](LICENSE) © Raunak Burrows.
