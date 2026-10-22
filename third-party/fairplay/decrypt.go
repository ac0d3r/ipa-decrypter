// High-level IPA decryption flow, ported from fairplay-decrypt's
// decrypt.py: scan a ZIP for SUPF-protected binaries, decrypt each with
// the account credential, and rewrite the archive with the results.
package fairplay

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ProgressFunc receives per-binary progress lines, e.g. "03/12 pages=842
// Payload/App.app/SC_Info/Binary". Mirrors decrypt.py's stdout prints.
type ProgressFunc func(line string)

// DecryptIPA decrypts the FairPlay-encrypted arm64 binaries inside ipaPath
// using credential (exactly 16 bytes) and writes the result to outputPath.
// The output path must not exist. It returns the total number of decrypted
// pages.
func DecryptIPA(ipaPath, outputPath string, credential []byte, block *BlockTransform, progress ProgressFunc) (int, error) {
	if len(credential) != 16 {
		return 0, fmt.Errorf("credential must contain exactly 16 bytes")
	}
	if _, err := os.Stat(ipaPath); err != nil {
		return 0, fmt.Errorf("input IPA missing: %w", err)
	}
	if _, err := os.Stat(outputPath); err == nil {
		return 0, fmt.Errorf("output already exists; never overwrite it")
	}
	if abs, err := filepath.Abs(outputPath); err == nil {
		if absIn, errIn := filepath.Abs(ipaPath); errIn == nil && abs == absIn {
			return 0, fmt.Errorf("input and output must differ")
		}
	}

	reader, err := zip.OpenReader(ipaPath)
	if err != nil {
		return 0, fmt.Errorf("open IPA: %w", err)
	}
	defer func() { _ = reader.Close() }()

	names := make(map[string]bool, len(reader.File))
	for _, f := range reader.File {
		names[f.Name] = true
	}
	if len(names) != len(reader.File) {
		return 0, fmt.Errorf("input ZIP has duplicate member names")
	}

	type planEntry struct {
		binaryPath string
		source     []byte
		sinf       []byte
		supf       []byte
	}
	var plan []planEntry
	for _, supfPath := range supfPaths(reader.File) {
		bundle, filename, ok := rsplit(supfPath, "/SC_Info/")
		if !ok {
			return 0, fmt.Errorf("SUPF outside SC_Info: %s", supfPath)
		}
		stem := filename
		isV5 := false
		if strings.HasSuffix(stem, ".v5.supf") {
			stem = strings.TrimSuffix(stem, ".v5.supf")
			isV5 = true
		} else {
			stem = strings.TrimSuffix(stem, ".supf")
		}
		binaryPath := bundle + "/" + stem
		sinfPath := bundle + "/SC_Info/" + stem + ".sinf"
		if isV5 && names[bundle+"/SC_Info/"+stem+".v4.sinf"] {
			sinfPath = bundle + "/SC_Info/" + stem + ".v4.sinf"
		}
		if !names[binaryPath] {
			return 0, fmt.Errorf("SUPF lacks binary/SINF: %s", supfPath)
		}
		source, err := readZipMember(&reader.Reader, binaryPath)
		if err != nil {
			return 0, err
		}
		if _, err := Arm64EncryptionInfo(source); err != nil {
			if strings.HasPrefix(err.Error(), "unsupported cryptid/size: 0,") {
				continue
			}
			return 0, err
		}
		if !names[sinfPath] {
			fallback := enclosingAppSinf(names, bundle)
			if fallback == "" {
				return 0, fmt.Errorf("SUPF lacks binary/SINF: %s", supfPath)
			}
			sinfPath = fallback
		}
		sinf, err := readZipMember(&reader.Reader, sinfPath)
		if err != nil {
			return 0, err
		}
		supf, err := readZipMember(&reader.Reader, supfPath)
		if err != nil {
			return 0, err
		}
		plan = append(plan, planEntry{
			binaryPath: binaryPath,
			source:     source,
			sinf:       sinf,
			supf:       supf,
		})
	}
	if len(plan) == 0 {
		return 0, fmt.Errorf("no encrypted arm64 binaries with supported SINF/SUPF")
	}

	replacements := make(map[string][]byte, len(plan))
	totalPages := 0
	for ordinal, entry := range plan {
		result, pages, err := DecryptBinary(entry.source, entry.sinf, entry.supf, credential, block)
		if err != nil {
			return 0, fmt.Errorf("%s: %w", entry.binaryPath, err)
		}
		replacements[entry.binaryPath] = result
		totalPages += pages
		if progress != nil {
			progress(fmt.Sprintf("%02d/%d pages=%d %s", ordinal+1, len(plan), pages, entry.binaryPath))
		}
	}

	if err := writeZip(&reader.Reader, replacements, outputPath); err != nil {
		return 0, err
	}
	return totalPages, nil
}

// rsplit splits s at the last occurrence of sep (Python's rsplit(sep, 1)).
func rsplit(s, sep string) (before, after string, ok bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return "", "", false
	}
	return s[:i], s[i+len(sep):], true
}

// supfPaths lists the SUPF member names, preferring the v5 variant when a
// binary has both.
func supfPaths(files []*zip.File) []string {
	byBinary := make(map[string]string)
	for _, f := range files {
		name := f.Name
		switch {
		case strings.HasSuffix(name, ".v5.supf"):
			byBinary[strings.TrimSuffix(name, ".v5.supf")] = name
		case strings.HasSuffix(name, ".supf"):
			stem := strings.TrimSuffix(name, ".supf")
			if _, ok := byBinary[stem]; !ok {
				byBinary[stem] = name
			}
		}
	}
	out := make([]string, 0, len(byBinary))
	for _, name := range byBinary {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// enclosingAppSinf finds a sibling .sinf inside the nearest enclosing .app
// bundle, used when the expected per-binary SINF is absent. Mirrors the
// updated fairplay-decrypt logic: .v4.sinf records are accepted, the
// unversioned variant is preferred, and a .v4.sinf sibling is tried before
// falling back to the first sorted record.
func enclosingAppSinf(names map[string]bool, bundle string) string {
	parts := strings.Split(bundle, "/")
	for i := len(parts); i > 0; i-- {
		prefix := strings.Join(parts[:i], "/")
		if !strings.HasSuffix(prefix, ".app") {
			continue
		}
		scInfo := prefix + "/SC_Info/"
		var sinfs []string
		for name := range names {
			if strings.HasPrefix(name, scInfo) &&
				strings.HasSuffix(name, ".sinf") &&
				strings.Count(name, "/") == strings.Count(scInfo, "/") {
				sinfs = append(sinfs, name)
			}
		}
		if len(sinfs) == 0 {
			continue
		}
		sort.Strings(sinfs)
		stem := stemOf(prefix)
		for _, suffix := range []string{".sinf", ".v4.sinf"} {
			if preferred := scInfo + stem + suffix; names[preferred] {
				return preferred
			}
		}
		var unversioned []string
		for _, name := range sinfs {
			if !strings.HasSuffix(name, ".v4.sinf") {
				unversioned = append(unversioned, name)
			}
		}
		if len(unversioned) > 0 {
			return unversioned[0]
		}
		return sinfs[0]
	}
	return ""
}

func stemOf(path string) string {
	base := path[strings.LastIndex(path, "/")+1:]
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return base
}

func readZipMember(r *zip.Reader, name string) ([]byte, error) {
	f, err := r.Open(name)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	defer func() { _ = f.Close() }()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", name, err)
	}
	return data, nil
}

// writeZip copies every member, swapping in decrypted replacements, and
// renames the provisional archive into place atomically.
func writeZip(r *zip.Reader, replacements map[string][]byte, outputPath string) error {
	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	provisional, err := os.CreateTemp(dir, "fairplay-output-*.ipa")
	if err != nil {
		return fmt.Errorf("create provisional output: %w", err)
	}
	provisionalName := provisional.Name()
	defer func() {
		_ = provisional.Close()
		_ = os.Remove(provisionalName) // no-op after successful rename
	}()

	writer := zip.NewWriter(provisional)
	for _, f := range r.File {
		header := f.FileHeader
		if f.FileInfo().IsDir() || strings.HasSuffix(f.Name, "/") {
			// Directory entries carry no data.
			if _, err := writer.CreateHeader(&header); err != nil {
				return fmt.Errorf("write %s: %w", f.Name, err)
			}
			continue
		}
		// Recompress every member; sizes and CRC are recomputed.
		header.CompressedSize64 = 0
		header.UncompressedSize64 = 0
		header.CRC32 = 0
		entry, err := writer.CreateHeader(&header)
		if err != nil {
			return fmt.Errorf("write %s: %w", f.Name, err)
		}
		var source io.Reader
		var member io.ReadCloser
		if data, ok := replacements[f.Name]; ok {
			source = bytes.NewReader(data)
		} else {
			m, err := r.Open(f.Name)
			if err != nil {
				return fmt.Errorf("read %s: %w", f.Name, err)
			}
			member = m
			source = member
		}
		if _, err := io.Copy(entry, source); err != nil {
			_ = member.Close()
			return fmt.Errorf("write %s: %w", f.Name, err)
		}
		if member != nil {
			if err := member.Close(); err != nil {
				return fmt.Errorf("close %s: %w", f.Name, err)
			}
		}
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("finalize output zip: %w", err)
	}
	if err := provisional.Close(); err != nil {
		return fmt.Errorf("flush output: %w", err)
	}
	if err := os.Rename(provisionalName, outputPath); err != nil {
		return fmt.Errorf("move output into place: %w", err)
	}
	return nil
}
