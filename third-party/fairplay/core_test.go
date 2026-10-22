package fairplay

import (
	"encoding/hex"
	"os"
	"path/filepath"
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
	root := os.Getenv("FAIRPLAY_ROOT")
	if root == "" {
		t.Skip("FAIRPLAY_ROOT not set; transform helper unavailable")
	}
	block, err := NewBlockTransform(
		filepath.Join(root, "transform"),
		filepath.Join(root, "transform_const_data.bin"),
	)
	if err != nil {
		t.Fatalf("start transform: %v", err)
	}
	defer block.Close()

	for i, v := range pageKeyVectors {
		got, err := DerivePageKey(decode16(t, v.wrapped), v.index, decode16(t, v.work), block)
		if err != nil {
			t.Fatalf("vector %d: %v", i, err)
		}
		if hex.EncodeToString(got) != v.want {
			t.Fatalf("vector %d: page key = %x, want %s", i, got, v.want)
		}
	}
}
