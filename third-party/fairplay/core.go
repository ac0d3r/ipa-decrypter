// FairPlay IPA decryption and page-key derivation.
//
// Go port of fairplay-decrypt's core.py. The block transform is pure Go
// (BlockTransform); no external helper is required.
package fairplay

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5" //nolint:gosec // G501: FairPlay SINF IV derivation requires MD5
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
)

const (
	arm64       = 0x0100000C
	mask32      = 0xFFFFFFFF
	mask64      = 0xFFFFFFFFFFFFFFFF
	pageSize    = 4096
	contextBase = 0x3E4A45B664EBA141
)

var (
	workMask = mustHex("c7626110cecd15963d850f6605f11c4b")
	constant = mustHex("b4f27cb84eaaff3c1309be4089f02972")

	roundSelectors = [][]byte{
		mustHex("363738393a3b3c3d"),
		mustHex("383736393a3d3c3b"),
		mustHex("38363a373c393d3b"),
		mustHex("37383a363d393b3c"),
	}
	roundConstants = [4]uint32{0x6EE6B74F, 0x78B72ABF, 0xD227C8EA, 0x0C0081BC}

	finalSelectors = [4][4]int{{0, 1, 2, 3}, {0, 2, 4, 6}, {0, 3, 5, 7}, {4, 5, 6, 7}}
	finalWords     = [4]uint32{0x85FE4668, 0x15EA8FCD, 0x3A17569B, 0x216E68E0}
)

func mustHex(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

// MachoInfo locates the encrypted region inside an arm64 Mach-O slice.
type MachoInfo struct {
	Base          int
	Cryptoff      int
	Cryptsize     int
	Cryptid       int
	CryptidOffset int
}

func xorBytes(a, b []byte) []byte {
	if len(a) != len(b) {
		panic("xor length mismatch")
	}
	out := make([]byte, len(a))
	for i := range a {
		out[i] = a[i] ^ b[i]
	}
	return out
}

// Arm64EncryptionInfo parses a (possibly FAT) Mach-O image and returns the
// single arm64 LC_ENCRYPTION_INFO record's location.
func Arm64EncryptionInfo(data []byte) (MachoInfo, error) {
	var arm [2]int // base, slice size
	switch {
	case bytes.Equal(data[:4], []byte{0xca, 0xfe, 0xba, 0xbe}):
		count := binary.BigEndian.Uint32(data[4:8])
		if count == 0 || count > 16 || 8+int(count)*20 > len(data) {
			return MachoInfo{}, fmt.Errorf("invalid FAT32 architecture table")
		}
		found := 0
		for i := 0; i < int(count); i++ {
			entry := data[8+i*20 : 8+(i+1)*20]
			cpu := binary.BigEndian.Uint32(entry[0:4])
			off := binary.BigEndian.Uint32(entry[8:12])
			size := binary.BigEndian.Uint32(entry[12:16])
			if cpu == arm64 {
				arm[0], arm[1] = int(off), int(size)
				found++
			}
		}
		if found != 1 {
			return MachoInfo{}, fmt.Errorf("expected one arm64 slice, found %d", found)
		}
	case bytes.Equal(data[:4], []byte{0xca, 0xfe, 0xba, 0xbf}):
		count := binary.BigEndian.Uint32(data[4:8])
		if count == 0 || count > 16 || 8+int(count)*32 > len(data) {
			return MachoInfo{}, fmt.Errorf("invalid FAT64 architecture table")
		}
		found := 0
		for i := 0; i < int(count); i++ {
			entry := data[8+i*32 : 8+(i+1)*32]
			cpu := binary.BigEndian.Uint32(entry[0:4])
			off := binary.BigEndian.Uint64(entry[8:16])
			size := binary.BigEndian.Uint64(entry[16:24])
			if cpu == arm64 {
				if off > math.MaxInt || size > math.MaxInt {
					return MachoInfo{}, fmt.Errorf("arm64 slice offset/size exceeds addressable range")
				}
				arm[0], arm[1] = int(off), int(size)
				found++
			}
		}
		if found != 1 {
			return MachoInfo{}, fmt.Errorf("expected one arm64 slice, found %d", found)
		}
	case bytes.Equal(data[:4], []byte{0xcf, 0xfa, 0xed, 0xfe}):
		arm[0], arm[1] = 0, len(data)
	default:
		return MachoInfo{}, fmt.Errorf("not a supported arm64 Mach-O")
	}
	base, sliceSize := arm[0], arm[1]
	if base+sliceSize > len(data) || base+32 > len(data) {
		return MachoInfo{}, fmt.Errorf("arm64 slice exceeds file")
	}
	header := data[base:]
	magic := binary.LittleEndian.Uint32(header[0:4])
	cpu := binary.LittleEndian.Uint32(header[4:8])
	ncmds := binary.LittleEndian.Uint32(header[16:20])
	sizeofcmds := binary.LittleEndian.Uint32(header[20:24])
	if magic != 0xFEEDFACF || cpu != arm64 || ncmds > 1000 {
		return MachoInfo{}, fmt.Errorf("invalid arm64 Mach-O header")
	}
	pos := base + 32
	commandsEnd := pos + int(sizeofcmds)
	if commandsEnd > base+sliceSize {
		return MachoInfo{}, fmt.Errorf("load commands exceed arm64 slice")
	}
	var found []MachoInfo
	for i := uint32(0); i < ncmds; i++ {
		if pos+8 > commandsEnd {
			return MachoInfo{}, fmt.Errorf("truncated load command")
		}
		cmd := binary.LittleEndian.Uint32(data[pos : pos+4])
		size := binary.LittleEndian.Uint32(data[pos+4 : pos+8])
		if size < 8 || pos+int(size) > commandsEnd {
			return MachoInfo{}, fmt.Errorf("invalid load command")
		}
		if cmd == 0x2C { // LC_ENCRYPTION_INFO
			if size < 24 {
				return MachoInfo{}, fmt.Errorf("short encryption info command")
			}
			cryptoff := binary.LittleEndian.Uint32(data[pos+8 : pos+12])
			cryptsize := binary.LittleEndian.Uint32(data[pos+12 : pos+16])
			cryptid := binary.LittleEndian.Uint32(data[pos+16 : pos+20])
			found = append(found, MachoInfo{
				Base:          base,
				Cryptoff:      int(cryptoff),
				Cryptsize:     int(cryptsize),
				Cryptid:       int(cryptid),
				CryptidOffset: pos + 16,
			})
		}
		pos += int(size)
	}
	if len(found) != 1 {
		return MachoInfo{}, fmt.Errorf("expected one encryption command, found %d", len(found))
	}
	info := found[0]
	if info.Cryptid != 1 || info.Cryptsize == 0 || info.Cryptsize%pageSize != 0 {
		return MachoInfo{}, fmt.Errorf("unsupported cryptid/size: %d, %d", info.Cryptid, info.Cryptsize)
	}
	if info.Cryptoff+info.Cryptsize > sliceSize || info.CryptidOffset >= base+info.Cryptoff {
		return MachoInfo{}, fmt.Errorf("invalid encrypted region bounds")
	}
	return info, nil
}

// sinfAtom is one SINF container field: a four-byte tag and its payload.
type sinfAtom struct {
	tag     []byte
	payload []byte
}

func sinfChildren(data []byte, start, end int) ([]sinfAtom, error) {
	var out []sinfAtom
	offset := start
	for offset < end {
		if offset+8 > end {
			return nil, fmt.Errorf("truncated SINF atom")
		}
		size := int(binary.BigEndian.Uint32(data[offset : offset+4]))
		if size < 8 || offset+size > end {
			return nil, fmt.Errorf("invalid SINF atom size at %#x", offset)
		}
		out = append(out, sinfAtom{
			tag:     data[offset+4 : offset+8],
			payload: data[offset+8 : offset+size],
		})
		offset += size
	}
	return out, nil
}

func sinfField(sinf, wanted []byte) ([]byte, error) {
	roots, err := sinfChildren(sinf, 0, len(sinf))
	if err != nil {
		return nil, err
	}
	if len(roots) != 1 || !bytes.Equal(roots[0].tag, []byte("sinf")) {
		return nil, fmt.Errorf("expected outer sinf atom")
	}
	schiChildren, err := sinfChildren(roots[0].payload, 0, len(roots[0].payload))
	if err != nil {
		return nil, err
	}
	var schis []sinfAtom
	for _, a := range schiChildren {
		if bytes.Equal(a.tag, []byte("schi")) {
			schis = append(schis, a)
		}
	}
	if len(schis) != 1 {
		return nil, fmt.Errorf("expected one schi container")
	}
	fields, err := sinfChildren(schis[0].payload, 0, len(schis[0].payload))
	if err != nil {
		return nil, err
	}
	var matched []sinfAtom
	for _, a := range fields {
		if bytes.Equal(a.tag, wanted) {
			matched = append(matched, a)
		}
	}
	if len(matched) != 1 {
		return nil, fmt.Errorf("expected one %q field", wanted)
	}
	return matched[0].payload, nil
}

func sinfAtomByName(sinf, name []byte) ([]byte, error) {
	tag := bytes.Index(sinf, name)
	if tag < 4 {
		return nil, fmt.Errorf("missing %q atom", name)
	}
	size := int(binary.BigEndian.Uint32(sinf[tag-4 : tag]))
	if size < 8 || tag-4+size > len(sinf) {
		return nil, fmt.Errorf("invalid %q atom size", name)
	}
	return sinf[tag+4 : tag-4+size], nil
}

// DeriveWork16 derives the 16-byte work key from the account credential
// and the binary's SINF record.
func DeriveWork16(credential, sinf []byte) ([]byte, error) {
	name, err := sinfField(sinf, []byte("name"))
	if err != nil {
		return nil, err
	}
	if i := bytes.IndexByte(name, 0); i >= 0 {
		name = name[:i]
	}
	iviv, err := sinfField(sinf, []byte("iviv"))
	if err != nil {
		return nil, err
	}
	rawPriv, err := sinfAtomByName(sinf, []byte("priv"))
	if err != nil {
		return nil, err
	}
	if len(rawPriv) != 440 && len(rawPriv) != 448 {
		return nil, fmt.Errorf("unsupported SINF priv-record length")
	}
	if len(rawPriv) == 440 {
		if !bytes.Equal(rawPriv[len(rawPriv)-8:], make([]byte, 8)) {
			return nil, fmt.Errorf("invalid SINF priv-record trailer")
		}
		rawPriv = rawPriv[:len(rawPriv)-8]
	}
	if len(credential) != 16 || len(iviv) != 16 {
		return nil, fmt.Errorf("invalid SINF priv-record dimensions")
	}
	iv := md5.Sum(append(append([]byte{}, name...), iviv...)) //nolint:gosec // G401: FairPlay SINF IV derivation requires MD5
	key := xorBytes(credential, bytes.Repeat([]byte("0"), 16))
	blockCipher, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(blockCipher, iv[:])
	plain := make([]byte, len(rawPriv))
	mode.CryptBlocks(plain, rawPriv)
	if !bytes.Equal(plain[:4], []byte("itun")) ||
		!bytes.Equal(plain[20:24], []byte("key ")) ||
		!bytes.Equal(plain[44:48], []byte("iviv")) {
		return nil, fmt.Errorf("credential does not decode this SINF priv record")
	}
	return xorBytes(plain[24:40], workMask), nil
}

// Arm64SUPFRecordOffset returns the byte offset of the arm64 page records
// inside a SUPF blob.
func Arm64SUPFRecordOffset(supf []byte, requiredPages int) (int, error) {
	if len(supf) < 12 || (supf[0] != 3 && supf[0] != 5) ||
		(!bytes.Equal(supf[1:4], []byte("510")) && !bytes.Equal(supf[1:4], []byte("612"))) {
		return 0, fmt.Errorf("unsupported SUPF format")
	}
	headerSize := 12
	if supf[0] == 3 {
		headerSize = 32
	}
	count := binary.BigEndian.Uint32(supf[8:12])
	if count == 0 || count > 16 {
		return 0, fmt.Errorf("invalid SUPF architecture count")
	}
	pos := 12
	for i := uint32(0); i < count; i++ {
		if pos+4+headerSize > len(supf) {
			return 0, fmt.Errorf("truncated SUPF architecture header")
		}
		length := binary.BigEndian.Uint32(supf[pos : pos+4])
		cpu := binary.BigEndian.Uint32(supf[pos+4 : pos+8])
		subtype := binary.BigEndian.Uint32(supf[pos+8 : pos+12])
		if int(length) < headerSize || pos+4+int(length) > len(supf) {
			return 0, fmt.Errorf("invalid SUPF architecture length")
		}
		pages := binary.BigEndian.Uint32(supf[pos+headerSize : pos+headerSize+4])
		if cpu == arm64 && subtype == 0 {
			if int(pages) < requiredPages || int(length) < headerSize+int(pages)*32 {
				return 0, fmt.Errorf("SUPF arm64 page records are incomplete")
			}
			return pos + 4 + headerSize, nil
		}
		pos += int(length) + 4
	}
	return 0, fmt.Errorf("arm64 SUPF record missing")
}

func high8FromContextWord(word uint64) []byte {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], word)

	fold := func(value, mask byte) byte {
		mixed := uint32(value) ^ uint32(mask)
		//nolint:gosec // G115: FairPlay high8 folding uses intentional 8-bit wrap
		return byte(mixed - (0x5A & (mixed << 1)) + 0x2D)
	}
	q := b[1] ^ 0xA1
	//nolint:gosec // G115: FairPlay high8 folding uses intentional 8-bit wrap
	seventh := byte(int32(q) - 2*int32((b[1]&0x52)^q) + 0x2D)
	return []byte{
		fold(b[7], 0x3E) ^ 0x3E,
		fold(b[6], 0x4A) ^ 0x24,
		fold(b[5], 0x45) ^ 0x93,
		fold(b[4], 0xB6) ^ 0x6D,
		fold(b[3], 0x64) ^ 0xA4,
		fold(b[2], 0xEB) ^ 0xDD,
		seventh ^ 0x04,
		(b[0] ^ 0x6C) ^ 0x5F,
	}
}

// SourceBlocks expands a wrapped SUPF page record and absolute page index
// into the three transform inputs.
func SourceBlocks(wrapped []byte, absoluteIndex uint32) ([3][]byte, error) {
	if len(wrapped) != 16 {
		return [3][]byte{}, fmt.Errorf("invalid wrapped record or page index")
	}
	context := contextBase ^ (uint64(absoluteIndex) << 12)
	first := append(xorBytes(wrapped[8:16], constant[:8]), high8FromContextWord(context)...)
	second := xorBytes(wrapped, constant)
	beHigh := binary.BigEndian.Uint64(constant[:8]) ^ (uint64(absoluteIndex) << 12)
	var prefix [8]byte
	binary.BigEndian.PutUint64(prefix[:], beHigh)
	third := append(prefix[:], xorBytes(wrapped[:8], constant[8:])...)
	return [3][]byte{first, second, third}, nil
}

func mixRound(before []uint16, selectors []byte) []uint16 {
	out := make([]uint16, 16)
	for pair := 0; pair < 4; pair++ {
		a := uint32(before[selectors[pair*2]-0x36]) ^ 0x18FB
		b := uint32(before[selectors[pair*2+1]-0x36]) ^ 0x18FB
		mixed := a + b
		square := (mixed*mixed ^ uint32(roundConstants[pair])) & mask32
		high := (mixed >> 16) & 0xFFFF
		low := ((0xCB61 - high*high) ^ square) & 0xFFFF
		top := (square >> 16) & 0xFFFF
		coded := top ^ 0x6B2
		out[pair*2] = uint16((low ^ 0x18FB) & 0xFFFF)
		out[pair*2+1] = uint16((coded - 2*((coded&0x58FF)^(top&0x4004)) + 0x18FB) & 0xFFFF)
	}
	return out
}

func finish(state []byte) ([]byte, error) {
	if len(state) != 16 {
		return nil, fmt.Errorf("finishing permutation requires 16 bytes")
	}
	initial := make([]uint16, 8)
	for i := 0; i < 16; i += 2 {
		initial[i/2] = binary.BigEndian.Uint16(state[i:i+2]) ^ 0x18FB
	}
	row := initial
	for _, selectors := range roundSelectors {
		row = mixRound(row, selectors)
	}
	final := make([]uint16, 0, 8)
	for j, selectors := range finalSelectors {
		d0 := row[selectors[0]] ^ 0x18FB
		d1 := row[selectors[1]] ^ 0x18FB
		d2 := row[selectors[2]] ^ 0x18FB
		d3 := row[selectors[3]] ^ 0x18FB
		left := uint64(d0) | uint64(d1)<<16
		right := uint64(d2) | uint64(d3)<<16
		x9 := left + right
		x19 := ((x9 & mask32) ^ 0x9CFD58B3DE3FD77F) & mask64
		x9 = (((x9 << 1) & 0x1BC7FAEFE) + x19 + 0x6302A74C21C02881) & mask64
		x9 = (x9 * x9) & mask64
		x19 = (x9 << 1) & 0x9E7B583F108F031C
		x9 = (x9 + 0x4F3DAC1F8847818E - x19) & mask64
		x19 = (uint64(finalWords[j]) + 0xD573525F) & mask32
		x22 := (x19 << 1) & 0x00000001108F031C
		x19 = (x19 + 0x4F3DAC1F8847818E - x22) & mask64
		x19 ^= x9
		x9 = x19 ^ (x9 >> 32)
		low := (x9 ^ 0xB4E4) & 0xFFFF
		high := ((x9 >> 16) ^ 0x4F3D) & mask32
		high = (high - ((high << 1) & 0x31F6) + 0x18FB) & 0xFFFF
		final = append(final, uint16(low), uint16(high))
	}
	out := make([]byte, 16)
	for i := 0; i < 8; i++ {
		binary.BigEndian.PutUint16(out[i*2:], initial[i]^final[i])
	}
	return out, nil
}

// DerivePageKey derives the AES key for one encrypted page.
func DerivePageKey(wrapped []byte, absoluteIndex uint32, work []byte) ([]byte, error) {
	sources, err := SourceBlocks(wrapped, absoluteIndex)
	if err != nil {
		return nil, err
	}
	transformed := make([][]byte, 3)
	for i, source := range sources {
		out, err := BlockTransform(source, work)
		if err != nil {
			return nil, err
		}
		transformed[i] = out
	}
	a, b, c := transformed[0], transformed[1], transformed[2]

	x1, err := finish(a)
	if err != nil {
		return nil, err
	}
	x2 := xorBytes(x1, b)
	x3, err := finish(x2)
	if err != nil {
		return nil, err
	}
	x4 := xorBytes(x3, c)
	return finish(x4)
}

// DecryptBinary decrypts every encrypted arm64 page of one binary and
// clears cryptid. It returns the patched binary and the page count.
func DecryptBinary(source, sinf, supf, credential []byte) (result []byte, pages int, err error) {
	info, err := Arm64EncryptionInfo(source)
	if err != nil {
		return nil, 0, err
	}
	pages = info.Cryptsize / pageSize
	recordStart, err := Arm64SUPFRecordOffset(supf, pages)
	if err != nil {
		return nil, 0, err
	}
	work, err := DeriveWork16(credential, sinf)
	if err != nil {
		return nil, 0, err
	}
	result = make([]byte, len(source))
	copy(result, source)
	basePage := (info.Base + info.Cryptoff) / pageSize
	for localPage := 0; localPage < pages; localPage++ {
		start := recordStart + 32*localPage
		record := supf[start : start+32]
		if len(record) != 32 {
			return nil, 0, fmt.Errorf("truncated SUPF page record")
		}
		pageIndex := basePage + localPage
		if pageIndex < 0 || pageIndex > math.MaxUint32 {
			return nil, 0, fmt.Errorf("page index exceeds uint32 range")
		}
		absoluteIndex := uint32(pageIndex) //nolint:gosec // G115: pageIndex bounded above
		key, err := DerivePageKey(record[:16], absoluteIndex, work)
		if err != nil {
			return nil, 0, err
		}
		pageStart := info.Base + info.Cryptoff + pageSize*localPage
		ciphertext := source[pageStart : pageStart+pageSize]
		blockCipher, err := aes.NewCipher(key)
		if err != nil {
			return nil, 0, err
		}
		mode := cipher.NewCBCDecrypter(blockCipher, record[16:32])
		page := make([]byte, pageSize)
		mode.CryptBlocks(page, ciphertext)
		copy(result[pageStart:pageStart+pageSize], page)
	}
	binary.LittleEndian.PutUint32(result[info.CryptidOffset:], 0)
	return result, pages, nil
}
