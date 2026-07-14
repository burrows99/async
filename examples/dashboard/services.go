package main

import (
	"fmt"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

// This file is the demo's "async function" layer.
//
// In JavaScript, the `async` keyword lives on the function definition, and
// calling the function hands back a Promise:
//
//	async function getUser(id) { ... }        // getUser(1) is a Promise<User>
//
// The Go equivalent is a function that returns *promise.Promise[T]. All the
// promise.New / closure wiring lives here — once per function, exactly where JS
// puts `async` — so the orchestration in main.go reads like the JS it mirrors,
// with no wrapping noise at the call sites.
//
// Plain, fast lookups use promise.New. Work that should stop when it is aborted
// (the mirrors, below) uses promise.WithSignal and watches its abort.Signal —
// the Go analogue of an async function that takes { signal } and passes it on.

type User struct {
	ID   int
	Name string
}

type Order struct {
	ID    int
	Total float64
}

type Recommendation struct {
	SKU   string
	Title string
}

func getUser(id int) *promise.Promise[User] {
	return promise.New(func() (User, error) {
		time.Sleep(40 * time.Millisecond)
		return User{ID: id, Name: fmt.Sprintf("user-%d", id)}, nil
	})
}

func getOrders(userID int) *promise.Promise[[]Order] {
	return promise.New(func() ([]Order, error) {
		time.Sleep(70 * time.Millisecond)
		return []Order{
			{ID: userID*100 + 1, Total: 42.00},
			{ID: userID*100 + 2, Total: 13.50},
		}, nil
	})
}

func getRecommendations() *promise.Promise[[]Recommendation] {
	return promise.New(func() ([]Recommendation, error) {
		time.Sleep(55 * time.Millisecond)
		return []Recommendation{
			{SKU: "GO-101", Title: "Go in Action"},
			{SKU: "GO-201", Title: "Concurrency in Go"},
		}, nil
	})
}

// getInventory always fails — it models a warehouse endpoint that is down, so
// the allSettled scene has a real partial failure to show.
func getInventory(sku string) *promise.Promise[int] {
	return promise.New(func() (int, error) {
		time.Sleep(30 * time.Millisecond)
		return 0, fmt.Errorf("inventory service unavailable for %s", sku)
	})
}

// getFromMirror is cancellable: it watches its signal, so a combinator (Race),
// Timeout, or an explicit Abort stops it promptly instead of leaving it running.
func getFromMirror(name string, latency time.Duration) *promise.Promise[string] {
	return promise.WithSignal(func(signal *abort.Signal) (string, error) {
		select {
		case <-time.After(latency):
			return fmt.Sprintf("payload from %s", name), nil
		case <-signal.Done():
			return "", signal.Reason()
		}
	})
}

// parsePositive panics on bad input — the Go analogue of a JS function that
// throws. promise.New contains the panic so it surfaces from Await as an error.
func parsePositive(n int) *promise.Promise[int] {
	return promise.New(func() (int, error) {
		if n < 0 {
			panic(fmt.Sprintf("parsePositive: expected a positive number, got %d", n))
		}
		return n, nil
	})
}
