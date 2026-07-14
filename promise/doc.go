// Package promise brings JavaScript's Promise vocabulary to Go: [New] mirrors
// `new Promise`, [Await] mirrors `await`, and [All], [Race], and [AllSettled]
// mirror the Promise combinators — while goroutines and context.Context run
// underneath.
//
//	// JS: const [user, orders] = await Promise.all([getUser(id), getOrders(id)]);
//	user, orders, err := promise.All2(getUser(id), getOrders(id))
//
// Callers never pass a context.Context. Each [Promise] owns an
// [github.com/burrows99/async/abort.Controller]; cancellation is reached through
// [Promise.Abort]. Because Go cannot preempt a goroutine, cancellation is opt-in
// exactly as in JavaScript: plain [New] work cannot be interrupted mid-flight,
// while work that wants to stop uses [WithSignal] and observes an
// abort.Signal — the Go analogue of threading { signal } into fetch.
//
// Await returns (T, error); there is no Catch. A panic inside a task is
// recovered and surfaced as a [*PanicError], so a task the package owns can
// never tear down the process. Errors compose with errors.Is and errors.As.
//
//go:generate go run github.com/burrows99/async/internal/gen
package promise
