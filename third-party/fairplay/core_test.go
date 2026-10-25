package fairplay

import (
	"encoding/hex"
	"testing"
)

func decode16(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil || len(b) != 16 {
		t.Fatalf("bad 16-byte vector %q", s)
	}
	return b
}

func decodeHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex vector %q", s)
	}
	return b
}

func TestFinish(t *testing.T) {
	for i, v := range finishVectors {
		got, err := finish(decode16(t, v.state))
		if err != nil {
			t.Fatalf("vector %d: %v", i, err)
		}
		if want := decode16(t, v.want); hex.EncodeToString(got) != v.want {
			t.Fatalf("vector %d: finish(%s) = %x, want %s", i, v.state, got, want)
		}
	}
}

func TestSourceBlocks(t *testing.T) {
	for i, v := range sourceBlockVectors {
		blocks, err := SourceBlocks(decode16(t, v.wrapped), v.index)
		if err != nil {
			t.Fatalf("vector %d: %v", i, err)
		}
		for j, got := range blocks {
			if hex.EncodeToString(got) != v.blocks[j] {
				t.Fatalf("vector %d block %d: = %x, want %s", i, j, got, v.blocks[j])
			}
		}
	}
}

func TestDeriveWork16(t *testing.T) {
	for i, v := range workVectors {
		got, err := DeriveWork16(decode16(t, v.credential), decodeHex(t, v.sinf))
		if err != nil {
			t.Fatalf("vector %d: %v", i, err)
		}
		if hex.EncodeToString(got) != v.want {
			t.Fatalf("vector %d: work = %x, want %s", i, got, v.want)
		}
	}
}

func TestDerivePageKey(t *testing.T) {
	for i, v := range pageKeyVectors {
		got, err := DerivePageKey(decode16(t, v.wrapped), v.index, decode16(t, v.work))
		if err != nil {
			t.Fatalf("vector %d: %v", i, err)
		}
		if hex.EncodeToString(got) != v.want {
			t.Fatalf("vector %d: page key = %x, want %s", i, got, v.want)
		}
	}
}

func TestBlockTransform(t *testing.T) {
	// AES-128-ECB encrypt of source ^ CONSTANT under work ^ WORK_MASK;
	// verified via page-key vectors and the end-to-end IPA test.
	if _, err := BlockTransform(make([]byte, 15), make([]byte, 16)); err == nil {
		t.Fatal("expected error for short source block")
	}
	if _, err := BlockTransform(make([]byte, 16), make([]byte, 15)); err == nil {
		t.Fatal("expected error for short work key")
	}
}
