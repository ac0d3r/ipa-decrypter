// Package fairplay decrypts FairPlay-encrypted IPAs with a 16-byte
// account credential. It is a Go port of fairplay-decrypt's core.py and
// decrypt.py; the block transform is pure Go and no longer shells out to
// an external helper.
package fairplay

import (
	"crypto/aes"
	"fmt"
)

// BlockTransform runs the FairPlay block transform: AES-128-ECB
// encryption of source ^ CONSTANT under the key work ^ WORK_MASK.
// Mirrors fairplay-decrypt's block_transform.
func BlockTransform(source, work []byte) ([]byte, error) {
	if len(source) != 16 || len(work) != 16 {
		return nil, fmt.Errorf("block-transform inputs require 16 bytes")
	}
	blockCipher, err := aes.NewCipher(xorBytes(work, workMask))
	if err != nil {
		return nil, fmt.Errorf("block transform cipher: %w", err)
	}
	out := make([]byte, 16)
	blockCipher.Encrypt(out, xorBytes(source, constant))
	return out, nil
}
