package store

import (
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrInvalidCursor indicates a malformed keyset cursor.
var ErrInvalidCursor = errors.New("store: invalid cursor")

// EncodeCursor encodes a (timestamp, id) keyset position into an opaque token.
func EncodeCursor(ts time.Time, id uuid.UUID) string {
	raw := ts.UTC().Format(time.RFC3339Nano) + "|" + id.String()
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

// DecodeCursor reverses EncodeCursor.
func DecodeCursor(s string) (time.Time, uuid.UUID, error) {
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	parts := strings.SplitN(string(b), "|", 2)
	if len(parts) != 2 {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	ts, err := time.Parse(time.RFC3339Nano, parts[0])
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	id, err := uuid.Parse(parts[1])
	if err != nil {
		return time.Time{}, uuid.Nil, ErrInvalidCursor
	}
	return ts, id, nil
}
