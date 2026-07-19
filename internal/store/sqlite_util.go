package store

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"

	idspkg "github.com/jeroenpfeil/mneme/internal/ids"
)

// This file holds the shared plumbing for the SQLite Store implementation:
// constraint-error classification, JSON (de)serialization for columns that are
// native types in Postgres (arrays, jsonb), app-side id/public_id minting, and
// float32 vector pack/unpack. Postgres relies on the driver and generated
// columns for these; SQLite does them in Go.

// rowScanner is the read side shared by *sql.Row and *sql.Rows, so one scan
// helper serves both Get (single row) and List (row iteration).
type rowScanner interface {
	Scan(dest ...any) error
}

// sqliteQB accumulates WHERE conditions and positional (?) args for the List
// queries — the SQLite analogue of the Postgres queryBuilder (which numbers its
// placeholders). Conditions carry their own "?" so callers stay explicit.
type sqliteQB struct {
	where []string
	args  []any
}

// add appends a condition and its bound argument(s). The condition must contain
// one "?" per arg.
func (b *sqliteQB) add(cond string, args ...any) {
	b.where = append(b.where, cond)
	b.args = append(b.args, args...)
}

func (b *sqliteQB) whereClause() string {
	if len(b.where) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(b.where, " AND ")
}

// limit appends a LIMIT clause when n > 0.
func (b *sqliteQB) limit(n int) string {
	if n > 0 {
		b.args = append(b.args, n)
		return " LIMIT ?"
	}
	return ""
}

// limitOffset appends LIMIT/OFFSET from a document Filter.
func (b *sqliteQB) limitOffset(f Filter) string {
	out := b.limit(f.Limit)
	if f.Offset > 0 {
		b.args = append(b.args, f.Offset)
		out += " OFFSET ?"
	}
	return out
}

// newUUID mints an internal primary-key value. Postgres uses gen_random_uuid();
// SQLite has no such default, so ids are supplied app-side on insert.
func newUUID() string { return uuid.NewString() }

// mintPublicID generates the portable public_id for a kind, matching the
// Postgres gen_public_id() format (prefix + Crockford-base32 body).
func mintPublicID(kind idspkg.Kind) (string, error) {
	id, err := idspkg.New(kind)
	if err != nil {
		return "", fmt.Errorf("mint public id: %w", err)
	}
	return id, nil
}

// isSQLiteFKViolation / isSQLiteUniqueViolation classify a modernc write error
// by its SQLite message (stable English), which names the exact failed
// constraint regardless of whether the primary or extended result code is
// surfaced. Used to translate raw errors into the store's typed domain errors.
func isSQLiteFKViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "FOREIGN KEY constraint failed")
}

func isSQLiteUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	m := err.Error()
	return strings.Contains(m, "UNIQUE constraint failed") ||
		strings.Contains(m, "PRIMARY KEY constraint failed")
}

// jsonArray encodes a string slice as a JSON array string for a TEXT column,
// normalizing nil to "[]" (SQLite has no native array type).
func jsonArray(v []string) string {
	b, _ := json.Marshal(ensureTags(v))
	return string(b)
}

// jsonObject encodes a map as a JSON object string for a TEXT column,
// normalizing nil to "{}".
func jsonObject(v map[string]any) (string, error) {
	b, err := json.Marshal(ensureJSONMap(v))
	if err != nil {
		return "", fmt.Errorf("marshal json object: %w", err)
	}
	return string(b), nil
}

// scanJSONArray decodes a JSON-array TEXT column back into a string slice,
// always returning a non-nil slice so it matches the Postgres array path.
func scanJSONArray(s string) ([]string, error) {
	out := []string{}
	if s == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("decode json array %q: %w", s, err)
	}
	if out == nil {
		out = []string{}
	}
	return out, nil
}

// scanJSONObject decodes a JSON-object TEXT column back into a map, always
// returning a non-nil map.
func scanJSONObject(s string) (map[string]any, error) {
	out := map[string]any{}
	if s == "" {
		return out, nil
	}
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, fmt.Errorf("decode json object %q: %w", s, err)
	}
	return out, nil
}

// packFloat32 serializes a vector to a little-endian float32 BLOB (4 bytes per
// element), the storage form of the embeddings.embedding column. nil/empty → nil.
func packFloat32(v []float32) []byte {
	if len(v) == 0 {
		return nil
	}
	b := make([]byte, 4*len(v))
	for i, f := range v {
		binary.LittleEndian.PutUint32(b[i*4:], math.Float32bits(f))
	}
	return b
}

// unpackFloat32 reverses packFloat32. A blob whose length is not a multiple of
// four is treated as corrupt and yields an error.
func unpackFloat32(b []byte) ([]float32, error) {
	if len(b) == 0 {
		return nil, nil
	}
	if len(b)%4 != 0 {
		return nil, fmt.Errorf("embedding blob length %d not a multiple of 4", len(b))
	}
	out := make([]float32, len(b)/4)
	for i := range out {
		out[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}
	return out, nil
}
