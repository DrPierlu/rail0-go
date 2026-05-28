package rail0

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GeneratePaymentID generates a checksummed RAIL0 payment ID (32 bytes,
// "0x"-prefixed hex).
//
// Layout:
//
//	bytes  0.. 3  — last 4 bytes of SHA-256(payload)   ← checksum
//	bytes  4..31  — 28 cryptographically-random bytes  ← payload
//
// The checksum lets Ponder (the on-chain indexer) verify that a payment was
// opened through rail0-api without a shared secret.  Any paymentId that
// fails the check is silently skipped by the indexer.
func GeneratePaymentID() (string, error) {
	payload := make([]byte, 28)
	if _, err := rand.Read(payload); err != nil {
		return "", fmt.Errorf("rail0: generate payment ID: %w", err)
	}

	digest := sha256.Sum256(payload)    // [32]byte
	checksum := digest[28:]             // last 4 bytes

	id := make([]byte, 32)
	copy(id[0:4], checksum)
	copy(id[4:], payload)

	return "0x" + hex.EncodeToString(id), nil
}
