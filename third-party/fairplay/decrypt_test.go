package fairplay

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// TestEndToEndIPADecrypt decrypts a real encrypted IPA (from
// fairplay-decrypt/out-account-a) and verifies the output matches the
// archive produced by the Python implementation. Skipped when the sample
// files are unavailable.
func TestEndToEndIPADecrypt(t *testing.T) {
	root := os.Getenv("FAIRPLAY_ROOT")
	if root == "" {
		t.Skip("FAIRPLAY_ROOT not set; sample IPA unavailable")
	}
	encrypted := filepath.Join(root, "out-account-a", "c45a6e7f-9120-425a-9e2a-f95b7126ba91.ipa")
	reference := filepath.Join(root, "out-account-a", "decrypted.ipa")
	credentialPath := filepath.Join(root, "out-account-a", "credential.bin")
	for _, path := range []string{encrypted, reference, credentialPath} {
		if _, err := os.Stat(path); err != nil {
			t.Skipf("sample file missing: %s", path)
		}
	}

	credential, err := os.ReadFile(credentialPath)
	if err != nil {
		t.Fatalf("read credential: %v", err)
	}

	outDir := t.TempDir()
	output := filepath.Join(outDir, "decrypted-go.ipa")
	pages, err := DecryptIPA(encrypted, output, credential, func(line string) {
		t.Logf("progress: %s", line)
	})
	if err != nil {
		t.Fatalf("DecryptIPA: %v", err)
	}
	if pages <= 0 {
		t.Fatalf("unexpected page count %d", pages)
	}

	// Deflate implementations differ, so compare decompressed member
	// content between the Go and Python archives instead of file digests.
	got, err := zipMemberDigests(output)
	if err != nil {
		t.Fatalf("read go output: %v", err)
	}
	want, err := zipMemberDigests(reference)
	if err != nil {
		t.Fatalf("read python output: %v", err)
	}
	if len(got) != len(want) {
		t.Fatalf("member count mismatch: go=%d python=%d", len(got), len(want))
	}
	for name, wantDigest := range want {
		gotDigest, ok := got[name]
		if !ok {
			t.Fatalf("member missing in go output: %s", name)
		}
		if gotDigest != wantDigest {
			t.Fatalf("member %s mismatch: go=%s python=%s", name, gotDigest, wantDigest)
		}
	}
}

// zipMemberDigests maps each member name to its decompressed SHA-256.
func zipMemberDigests(path string) (map[string]string, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	out := make(map[string]string, len(reader.File))
	for _, f := range reader.File {
		member, err := f.Open()
		if err != nil {
			return nil, err
		}
		digest := sha256.New()
		if _, err := io.Copy(digest, member); err != nil {
			member.Close()
			return nil, err
		}
		member.Close()
		out[f.Name] = hex.EncodeToString(digest.Sum(nil))
	}
	return out, nil
}
