package store

import (
	"math"
	"testing"
)

// TestFloat32BlobRoundTrip verifies the embedding BLOB codec preserves every
// value bit-for-bit — the SQLite backend stores pgvector's vector(1024) as a
// packed little-endian float32 blob and reads it back for brute-force cosine.
func TestFloat32BlobRoundTrip(t *testing.T) {
	in := []float32{0, 1, -1, 0.5, -0.25, 3.14159, math.MaxFloat32, math.SmallestNonzeroFloat32}
	blob := packFloat32(in)
	if len(blob) != 4*len(in) {
		t.Fatalf("packed length %d, want %d", len(blob), 4*len(in))
	}
	out, err := unpackFloat32(blob)
	if err != nil {
		t.Fatalf("unpackFloat32: %v", err)
	}
	if len(out) != len(in) {
		t.Fatalf("unpacked length %d, want %d", len(out), len(in))
	}
	for i := range in {
		if out[i] != in[i] {
			t.Errorf("index %d: got %v, want %v", i, out[i], in[i])
		}
	}
}

func TestFloat32BlobEdgeCases(t *testing.T) {
	if b := packFloat32(nil); b != nil {
		t.Errorf("packFloat32(nil): got %v, want nil", b)
	}
	if b := packFloat32([]float32{}); b != nil {
		t.Errorf("packFloat32(empty): got %v, want nil", b)
	}
	if v, err := unpackFloat32(nil); v != nil || err != nil {
		t.Errorf("unpackFloat32(nil): got %v, %v", v, err)
	}
	if _, err := unpackFloat32([]byte{1, 2, 3}); err == nil {
		t.Error("unpackFloat32(len 3): want error for non-multiple-of-4")
	}
}
