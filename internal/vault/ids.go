package vault

import (
	"crypto/rand"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewID returns a fresh ULID string. ULIDs sort lexicographically by
// creation time, which keeps note ordering stable and makes id collisions
// vanishingly unlikely (128 bits, 80 of them random).
func NewID() string {
	return ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader).String()
}

// ValidID reports whether s parses as a ULID. Used at load time to reject
// notes with hand-typed or truncated ids.
func ValidID(s string) bool {
	_, err := ulid.ParseStrict(s)
	return err == nil
}
