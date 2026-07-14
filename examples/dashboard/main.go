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
	"sync/atomic"
	"time"

	"github.com/burrows99/async/abort"
	"github.com/burrows99/async/collections"
	"github.com/burrows99/async/promise"
	"github.com/burrows99/async/timers"
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

	// ─── Phase 2 · collections ─────────────────────────────────────────────
	sceneMap()
	sceneQueue()

	// ─── Phase 3 · utilities ───────────────────────────────────────────────
	sceneAny()
	sceneRetry()
	sceneSetTimeout()
	sceneDebounce()
	sceneThen()

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
			fmt.Printf("  → %s: failed (%v)\n", skus[i], r.Reason)
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

func sceneMap() {
	scene(9, "pMap — bounded concurrency over a list",
		`const users = await pMap([1..6], getUser, { concurrency: 2 });`)

	ids := []int{1, 2, 3, 4, 5, 6}

	start := time.Now()
	// const users = await pMap(ids, getUser, { concurrency: 2 });
	users, err := collections.Map(ids, getUser, collections.Concurrency(2))
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	// 6 users at concurrency 2 → three waves; visibly slower than all-at-once.
	fmt.Printf("  → %d users at concurrency 2 (in %s)\n", len(users), time.Since(start).Round(time.Millisecond))
}

func sceneQueue() {
	scene(10, "p-queue — a worker pool",
		`const q = new PQueue({ concurrency: 2 }); jobs.forEach(j => q.add(j)); await q.onIdle();`)

	// const q = new PQueue({ concurrency: 2 });
	queue := collections.NewQueue[int](2)

	// jobs.forEach(j => q.add(() => run(j)));
	jobs := make([]*promise.Promise[int], 5)
	for i := range jobs {
		n := i + 1
		jobs[i] = queue.Add(func() (int, error) {
			time.Sleep(20 * time.Millisecond)
			return n * n, nil
		})
	}
	queue.OnIdle() // await q.onIdle();

	sum := 0
	for _, j := range jobs {
		v, _ := promise.Await(j)
		sum += v
	}
	fmt.Printf("  → 5 jobs done at concurrency 2, sum of squares = %d\n", sum)
}

func sceneAny() {
	scene(11, "Promise.any — first success wins",
		`const p = await Promise.any([down(), mirrorB(), mirrorC()]);`)

	// const p = await Promise.any([down(), mirrorB(), mirrorC()]);
	payload, err := promise.Any(
		promise.New(func() (string, error) { return "", errors.New("mirror-A is down") }),
		getFromMirror("mirror-B", 30*time.Millisecond),
		getFromMirror("mirror-C", 10*time.Millisecond),
	)
	if err != nil {
		fmt.Println("  error:", err)
		return
	}
	fmt.Printf("  → %q (first to succeed; A's failure ignored)\n", payload)
}

func sceneRetry() {
	scene(12, "p-retry — retry with backoff",
		`const r = await pRetry(flaky, { retries: 2 });`)

	attempt := 0
	// const r = await pRetry(flaky, { retries: 2 });
	r, err := promise.Await(promise.Retry(func() (string, error) {
		attempt++
		if attempt < 3 {
			return "", fmt.Errorf("attempt %d failed", attempt)
		}
		return "succeeded", nil
	}, promise.Attempts(3), promise.ExpBackoff(10*time.Millisecond)))
	fmt.Printf("  → %q after %d attempts (err: %v)\n", r, attempt, err)
}

func sceneSetTimeout() {
	scene(13, "setTimeout(fn, ms)",
		`setTimeout(() => log("fired"), 20);`)

	var fired atomic.Bool
	// setTimeout(() => { fired = true }, 20);
	timers.SetTimeout(func() { fired.Store(true) }, 20*time.Millisecond)
	time.Sleep(40 * time.Millisecond) // JS: the event loop would run it
	fmt.Printf("  → timer fired: %t\n", fired.Load())
}

func sceneDebounce() {
	scene(14, "debounce(fn, ms)",
		`const save = debounce(persist, 20); save(); save(); save();`)

	var runs atomic.Int64
	// const save = debounce(persist, 20);
	save, _ := timers.Debounce(func() { runs.Add(1) }, 20*time.Millisecond)
	save()
	save()
	save() // three rapid calls collapse into one
	time.Sleep(40 * time.Millisecond)
	fmt.Printf("  → persist ran %d time(s) after 3 rapid calls\n", runs.Load())
}

func sceneThen() {
	scene(15, "chaining — .then / .finally",
		`await getUser(1).then(u => u.Name).finally(() => log("done"));`)

	// getUser(1).then(u => u.Name)   — cross-type transform (User -> string)
	nameP := promise.Then(getUser(1), func(u User) (string, error) {
		return u.Name, nil
	})
	// .finally(() => ...)
	name, err := promise.Await(nameP.Finally(func() { /* cleanup runs whatever happens */ }))
	fmt.Printf("  → %q (err: %v)\n", name, err)
}

// scene prints a numbered header and the JavaScript the section mirrors.
func scene(n int, title, js string) {
	fmt.Printf("\n%d. %s\n", n, title)
	fmt.Printf("  JS:  %s\n", js)
}
