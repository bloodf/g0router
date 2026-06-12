package usage

import "sync"

// Events is a mutex-guarded synchronous event fan-out registry.
// It mirrors the statsEmitter EventEmitter from the reference implementation.
type Events struct {
	mu    sync.Mutex
	cbs   []func(kind string)
}

// NewEvents creates an empty event emitter.
func NewEvents() *Events {
	return &Events{}
}

// OnEvent registers a callback that is invoked synchronously for each Emit.
func (e *Events) OnEvent(fn func(kind string)) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.cbs = append(e.cbs, fn)
}

// Emit invokes all registered callbacks with the given kind.
func (e *Events) Emit(kind string) {
	e.mu.Lock()
	cbs := make([]func(kind string), len(e.cbs))
	copy(cbs, e.cbs)
	e.mu.Unlock()

	for _, cb := range cbs {
		cb(kind)
	}
}
