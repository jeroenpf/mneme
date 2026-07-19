package embed

import "sync"

// pendingSet is the worker's deduplicated, non-dropping work queue. Repeat
// enqueues of the same source coalesce (a source waiting to be embedded is
// held once, and re-processed with its latest state), and it never drops work
// under bursts — unlike the old fixed-size channel. Restart durability is
// provided separately by reconciliation. Safe for concurrent use.
type pendingSet struct {
	mu    sync.Mutex
	set   map[SourceRef]struct{}
	order []SourceRef // FIFO of the distinct refs currently pending
}

func newPendingSet(hint int) *pendingSet {
	if hint < 0 {
		hint = 0
	}
	return &pendingSet{
		set:   make(map[SourceRef]struct{}, hint),
		order: make([]SourceRef, 0, hint),
	}
}

// Add enqueues ref, returning true iff it was not already pending (a genuinely
// new item worth signalling a waiter about).
func (p *pendingSet) Add(ref SourceRef) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if _, ok := p.set[ref]; ok {
		return false
	}
	p.set[ref] = struct{}{}
	p.order = append(p.order, ref)
	return true
}

// Next pops the oldest pending ref, returning ok=false when the set is empty.
// A popped ref leaves the set, so a later change to the same source re-enqueues.
func (p *pendingSet) Next() (SourceRef, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.order) == 0 {
		return SourceRef{}, false
	}
	ref := p.order[0]
	p.order = p.order[1:]
	delete(p.set, ref)
	return ref, true
}

// Len reports how many distinct refs are currently pending.
func (p *pendingSet) Len() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.order)
}
