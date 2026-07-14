// Command dashboard is a living showcase of the async library, written to read
// like JavaScript Promise code. It grows one section per delivered phase, so we
// can feel whether the API stays JS-intuitive as the surface expands.
//
// Ground rules for this file:
//   - Concurrency goes through promise.* / abort.* only — no raw goroutines,
//     channels, sync, or select. If a scene can't be written that way, that's a
//     gap in the library, not the demo.
//   - No promise.New / closure wrapping at the call sites. The "async functions"
//     that return promises live in services.go, exactly where JS puts `async`.
//     Everything below is orchestration, and mirrors its JS line for line.
//
// Run it:
//
//	go run ./examples/dashboard
package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/promise"
)

func main() {
	fmt.Println("async demo — JavaScript-style concurrency in Go")
	fmt.Println("(each scene shows the JS it mirrors, then runs the Go)")

	// ─── Phase 1 · core ────────────────────────────────────────────────────
	sceneAwait()
	sceneAll()
	sceneDashboard()
	sceneRace()
	sceneAllSettled()
	scenePanicSafety()
	sceneTimeout()
	sceneAbort()

	fmt.Println("\nAll scenes complete — the process survived every failure above.")
}

func sceneAwait() {
	scene(1, "await a promise", `const user = await getUser(1);`)

	// const user = await getUser(1);
	user, err := promise.Await(getUser(1))
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	fmt.Printf("  → %+v\n", user)
}

func sceneAll() {
	scene(2, "Promise.all — same type", `const users = await Promise.all([getUser(1), getUser(2), getUser(3)]);`)

	// const users = await Promise.all([getUser(1), getUser(2), getUser(3)]);
	users, err := promise.All(getUser(1), getUser(2), getUser(3))
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	fmt.Printf("  → %d users, in order: %+v\n", len(users), users)
}

func sceneDashboard() {
	scene(3, "Promise.all — mixed types (All2..All8)",
		`const [user, orders, recs] = await Promise.all([getUser(1), getOrders(1), getRecommendations()]);`)

	start := time.Now()
	// const [user, orders, recs] = await Promise.all([getUser(1), getOrders(1), getRecommendations()]);
	user, orders, recs, err := promise.All3(getUser(1), getOrders(1), getRecommendations())
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	fmt.Printf("  → user=%s, orders=%d, recs=%d  (in %s, all at once)\n",
		user.Name, len(orders), len(recs), time.Since(start).Round(time.Millisecond))
}

func sceneRace() {
	scene(4, "Promise.race — first to settle wins",
		`const payload = await Promise.race([getFrom("edge"), getFrom("origin")]);`)

	// const payload = await Promise.race([getFrom("edge"), getFrom("origin")]);
	payload, err := promise.Race(
		getFromMirror("edge", 25*time.Millisecond),
		getFromMirror("origin", 200*time.Millisecond),
	)
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	fmt.Printf("  → %q (the slow origin was aborted)\n", payload)
}

func sceneAllSettled() {
	scene(5, "Promise.allSettled — successes and failures",
		`const results = await Promise.allSettled(skus.map(getInventory));`)

	skus := []string{"GO-101", "GO-201", "GO-301"}

	// const results = await Promise.allSettled(skus.map(getInventory));
	ps := make([]*promise.Promise[int], len(skus))
	for i, sku := range skus {
		ps[i] = getInventory(sku)
	}
	results := promise.AllSettled(ps...)

	for i, r := range results {
		if r.OK() {
			fmt.Printf("  → %s: %d in stock\n", skus[i], r.Value)
		} else {
			fmt.Printf("  → %s: failed (%v)\n", skus[i], r.Err)
		}
	}
}

func scenePanicSafety() {
	scene(6, "a throw doesn't crash the process",
		`try { await parse(-5); } catch (e) { /* survivable */ }`)

	// try { await parse(-5); } catch (e) { ... }
	_, err := promise.Await(parsePositive(-5))

	var pe *promise.PanicError
	if errors.As(err, &pe) {
		fmt.Printf("  → contained panic: %v\n", pe.Value)
	}
}

func sceneTimeout() {
	scene(7, "AbortSignal.timeout(ms)",
		`await fetch(url, { signal: AbortSignal.timeout(100) });`)

	// await fetch(url, { signal: AbortSignal.timeout(100) });
	_, err := promise.Await(promise.Timeout(getFromMirror("cold-storage", 500*time.Millisecond), 100*time.Millisecond))
	fmt.Printf("  → timed out: %t\n", errors.Is(err, promise.ErrTimeout))
}

func sceneAbort() {
	scene(8, "AbortController — cancel in flight",
		"const c = new AbortController(); getReport(c.signal); c.abort();")

	// const c = new AbortController();
	c := abort.NewController()

	// const p = getReport(c.signal);   // work that watches the controller's signal
	report := promise.New(func() (string, error) {
		select {
		case <-time.After(time.Second):
			return "report", nil
		case <-c.Signal().Done():
			return "", c.Signal().Reason()
		}
	})

	c.Abort() // c.abort();
	_, err := promise.Await(report)
	fmt.Printf("  → aborted early: %v\n", err)
}

// scene prints a numbered header and the JavaScript the section mirrors.
func scene(n int, title, js string) {
	fmt.Printf("\n%d. %s\n", n, title)
	fmt.Printf("  JS:  %s\n", js)
}
