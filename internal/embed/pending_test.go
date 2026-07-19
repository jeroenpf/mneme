package embed

import "testing"

func TestPendingSetDedup(t *testing.T) {
	p := newPendingSet(0)
	ref := SourceRef{Type: "documents", ID: "d1"}
	if !p.Add(ref) {
		t.Fatal("first Add should report the ref as newly pending")
	}
	if p.Add(ref) {
		t.Fatal("re-adding a still-pending ref should report already pending")
	}
	if p.Len() != 1 {
		t.Fatalf("dedup should keep a single entry, got %d", p.Len())
	}
}

func TestPendingSetFIFOAndReadd(t *testing.T) {
	p := newPendingSet(0)
	a := SourceRef{Type: "documents", ID: "a"}
	b := SourceRef{Type: "decisions", ID: "b"}
	p.Add(a)
	p.Add(b)

	got1, ok1 := p.Next()
	got2, ok2 := p.Next()
	_, ok3 := p.Next()
	if !ok1 || got1 != a {
		t.Fatalf("first Next should pop a (FIFO), got %+v ok=%v", got1, ok1)
	}
	if !ok2 || got2 != b {
		t.Fatalf("second Next should pop b, got %+v ok=%v", got2, ok2)
	}
	if ok3 {
		t.Fatal("Next on an empty set should report false")
	}
	// Once popped, a ref may be enqueued again (a fresh change after dequeue).
	if !p.Add(a) {
		t.Fatal("re-adding a after it was popped should report newly pending")
	}
}

func TestPendingSetNeverDrops(t *testing.T) {
	p := newPendingSet(4) // small hint; must still hold everything
	const n = 500
	for i := 0; i < n; i++ {
		p.Add(SourceRef{Type: "documents", ID: string(rune('a' + i%26)) + string(rune('0'+i/26))})
	}
	if p.Len() != n {
		t.Fatalf("non-dropping queue should hold all %d distinct refs, got %d", n, p.Len())
	}
}
