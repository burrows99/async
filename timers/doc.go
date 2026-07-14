// Package timers brings JavaScript's timing functions to Go, with the same
// names and argument order: [SetTimeout] and [SetInterval] (with [ClearTimeout]
// and [ClearInterval] to cancel), plus lodash's [Debounce] and [Throttle].
//
//	// JS: const id = setTimeout(() => save(), 200); clearTimeout(id);
//	t := timers.SetTimeout(save, 200*time.Millisecond)
//	timers.ClearTimeout(t)
//
// Like their JavaScript counterparts, these take no context: the returned handle
// (or the cancel function from Debounce/Throttle) is how you stop them.
package timers
