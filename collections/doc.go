// Package collections runs a function over a slice concurrently — the Go
// analogue of Array.map followed by Promise.all, or the p-map / p-queue
// libraries from npm.
//
//	// JS: const users = await pMap(ids, getUser, { concurrency: 10 });
//	users, err := collections.Map(ids, getUser, collections.Concurrency(10))
//
// [Map] and [ForEach] map an "async function" (one returning
// *[github.com/burrows99/async/promise.Promise]) over a slice, optionally
// bounding how many run at once with [Concurrency]. [Queue] is a reusable
// work queue with the same bound (npm's p-queue). All of them are fail-fast like
// Promise.all: the first error stops new work from starting and is returned,
// while in-flight work — which Go, like JavaScript, cannot preempt — runs to
// completion.
package collections
