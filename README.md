# async

[![Go Reference](https://pkg.go.dev/badge/github.com/burrows99/async.svg)](https://pkg.go.dev/github.com/burrows99/async)
[![Go Report Card](https://goreportcard.com/badge/github.com/burrows99/async)](https://goreportcard.com/report/github.com/burrows99/async)

JavaScript-style concurrency for Go. `async` brings the vocabulary you already
know — `Promise`, `await`, `Promise.all`, `AbortController` — to Go, so
orchestration code reads like the JS it mirrors while goroutines and
`context.Context` run underneath.

```go
// JS: const [user, orders, recs] = await Promise.all([getUser(1), getOrders(1), getRecs(1)]);
user, orders, recs, err := promise.All3(getUser(1), getOrders(1), getRecs(1))
```

It is an **onboarding ramp and a boilerplate-killer for the 90% case** — not a
replacement for channels. When you outgrow it, drop back to goroutines and
`select`; `async` only accepts and returns standard types, so it interoperates
rather than competes.

> **Status: core (v0.1) — in development.** The surface below is implemented,
> tested with the race detector, and documented. Collections (`Map`, `Pool`, …)
> and utilities (`Retry`, `Debounce`, …) land in later phases. See [Roadmap](#roadmap).

## Install

```sh
go get github.com/burrows99/async
```

Requires Go 1.23+. Zero dependencies beyond the standard library.

## Packages

The library is organized by JavaScript concept, so imports read like the API
they provide:

| Package | Mirrors | Provides |
|---------|---------|----------|
| [`promise`](promise) | `Promise` | `New`, `WithSignal`, `Await`, `All`, `All2`–`All8`, `Race`, `AllSettled`, `Timeout` |
| [`abort`](abort) | `AbortController` / `AbortSignal` | `NewController`, `Controller`, `Signal` |

```go
import (
    "github.com/burrows99/async/promise"
    "github.com/burrows99/async/abort"
)
```

## Quick start

Put the `promise.New` wrapping in your function definitions — exactly where JS
puts the `async` keyword — and return a `*promise.Promise[T]`. Then every call
site is clean:

```go
// This is our "async function". Calling it returns a Promise, just like
// `async function getUser(id) { ... }` does in JS.
func getUser(id int) *promise.Promise[User] {
    return promise.New(func() (User, error) {
        return db.FindUser(id)
    })
}
```

```go
// JS: const user = await getUser(1);
user, err := promise.Await(getUser(1))

// JS: const users = await Promise.all([getUser(1), getUser(2), getUser(3)]);
users, err := promise.All(getUser(1), getUser(2), getUser(3))

// JS: const [user, orders] = await Promise.all([getUser(1), getOrders(1)]);
user, orders, err := promise.All2(getUser(1), getOrders(1))
```

See [`examples/dashboard`](examples/dashboard) for a full, runnable showcase:

```sh
go run ./examples/dashboard
```

## Cancellation is opt-in — like JavaScript

Go cannot preempt a goroutine, and neither can JavaScript preempt a running
function. That is why JS cancellation is opt-in: you thread an `AbortSignal`
into `fetch(url, { signal })`, and work that ignores the signal simply keeps
running. `async` works the same way.

- **Plain work** uses `promise.New(func() (T, error))` — no signal. It cannot be
  interrupted mid-flight; `Abort` marks it aborted for awaiters, but the function
  runs to completion.
- **Cancellable work** uses `promise.WithSignal(func(signal *abort.Signal) (T, error))`
  and watches the signal — the Go analogue of accepting `{ signal }`.

```go
func getFromMirror(name string) *promise.Promise[string] {
    return promise.WithSignal(func(signal *abort.Signal) (string, error) {
        select {
        case <-time.After(latency):
            return payload, nil
        case <-signal.Done():          // aborted by Abort, a combinator, or Timeout
            return "", signal.Reason() // abort.ErrAborted
        }
    })
}
```

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

## Errors and panics

- **Errors, not rejections.** `Await` returns `(T, error)`; there is no `.Catch`.
  Orchestration stays JS-shaped, error handling stays Go-shaped and
  `errors.Is`/`errors.As`-compatible.
- **A panic never crashes the process.** Every function runs under `recover()`; a
  panic becomes a `*promise.PanicError` carrying the value and a stack trace,
  returned from `Await` like any other error.

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
- **Collections:** `collections.Map`, `ForEach`, `WithLimit`, `Pool`.
- **Utilities:** `Retry`, `timers.SetTimeout`/`SetInterval`, `Debounce`,
  `Throttle`, `Any` + `AggregateError`, plus a "coming from JavaScript"
  migration guide.

## Contributing

The `All2`…`All8` variants in `promise/alln.go` are generated — Go generics have
no variadic type parameters. Edit `internal/gen`, then regenerate:

```sh
go generate ./...
```

Before sending a change:

```sh
go test -race ./...
go vet ./...
gofmt -l .
```

## License

[MIT](LICENSE) © Raunak Burrows
