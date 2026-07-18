// Package ids mints and validates Mneme's stable public identifiers.
//
// Three layers of identity are kept deliberately distinct:
//
//   - The public id — what this package mints. A short opaque string of the
//     form "<prefix>_<body>", where body is twelve lower-case Crockford
//     Base32 characters (60 bits of entropy). It is the one identity that is
//     stable and portable: safe to paste into a reference and to store as a
//     foreign key. Every addressable entity has exactly one — project,
//     document, block, task, decision, journal, snippet, solution.
//   - The internal database id — the storage engine's own row key (a UUID or
//     serial). It never leaves the store and is not an addressing surface.
//   - The mutable slug and title — human-facing labels (a document's
//     kebab-case slug, an entity's title). They change with edits and moves,
//     so they are display and search affordances, never identity.
//
// A public id is therefore independent of any title or slug and survives
// edits, renames, and moves. This is a leaf package: it imports nothing else
// in Mneme, so both the store and the MCP layer can depend on it.
//
// Security doctrine: a public id is an identifier, never an access-control
// token. Its opacity is not a secret and knowing one grants no authority —
// resolving a reference must run the same access checks as any normal read.
// Do not gate authorization on possession of an id.
package ids

import (
	"crypto/rand"
	"fmt"
	"io"
	"strings"
)

// Kind is an addressable entity kind. Its string value doubles as the
// segment used in a canonical mneme:// reference (see refs.go).
type Kind string

const (
	KindProject  Kind = "project"
	KindDocument Kind = "document"
	KindBlock    Kind = "block"
	KindTask     Kind = "task"
	KindDecision Kind = "decision"
	KindJournal  Kind = "journal"
	KindSnippet  Kind = "snippet"
	KindSolution Kind = "solution"
)

// bodyLen is the number of Base32 characters in an id body: 12 chars ×
// 5 bits = 60 bits of entropy, ample against collisions at personal scale.
const bodyLen = 12

// alphabet is Crockford Base32, lower-cased, excluding i, l, o, u to stay
// unambiguous when read or transcribed. Exactly 32 symbols, so five bits
// of randomness map to one character with no modulo bias.
const alphabet = "0123456789abcdefghjkmnpqrstvwxyz"

// kindPrefix registers the id prefix (without the underscore) for each
// kind. Prefixes are reserved: Parse rejects any id whose prefix is not in
// this registry.
var kindPrefix = map[Kind]string{
	KindProject:  "prj",
	KindDocument: "doc",
	KindBlock:    "blk",
	KindTask:     "task",
	KindDecision: "dec",
	KindJournal:  "jrnl",
	KindSnippet:  "snip",
	KindSolution: "sol",
}

// prefixKind is the reverse of kindPrefix, built once at init.
var prefixKind = func() map[string]Kind {
	m := make(map[string]Kind, len(kindPrefix))
	for k, p := range kindPrefix {
		m[p] = k
	}
	return m
}()

// maxRetries bounds NewUnique's collision-retry loop. Reaching it means the
// random source is broken or the exists predicate always returns true —
// never real 60-bit collisions.
const maxRetries = 8

// Gen mints ids from an injectable random source. Construct one with NewGen
// for deterministic tests; the package-level New/NewUnique use crypto/rand.
type Gen struct {
	r io.Reader
}

// NewGen returns a generator drawing randomness from r.
func NewGen(r io.Reader) *Gen { return &Gen{r: r} }

// defaultGen is the crypto/rand-backed generator behind the package funcs.
var defaultGen = &Gen{r: rand.Reader}

// New mints a fresh id for kind. It errors on an unregistered kind or a
// random-source read failure.
func (g *Gen) New(kind Kind) (string, error) {
	prefix, ok := kindPrefix[kind]
	if !ok {
		return "", fmt.Errorf("ids: unknown kind %q", kind)
	}
	body, err := g.body()
	if err != nil {
		return "", err
	}
	return prefix + "_" + body, nil
}

// NewUnique mints an id for kind that fails the exists predicate, retrying
// on collision. A nil predicate means "nothing is taken". It errors after
// maxRetries consecutive collisions.
func (g *Gen) NewUnique(kind Kind, exists func(string) bool) (string, error) {
	for i := 0; i < maxRetries; i++ {
		id, err := g.New(kind)
		if err != nil {
			return "", err
		}
		if exists == nil || !exists(id) {
			return id, nil
		}
	}
	return "", fmt.Errorf("ids: no free %s id after %d attempts", kind, maxRetries)
}

// body reads bodyLen random bytes and maps each to one alphabet symbol via
// its low five bits — uniform because 32 divides 256.
func (g *Gen) body() (string, error) {
	buf := make([]byte, bodyLen)
	if _, err := io.ReadFull(g.r, buf); err != nil {
		return "", fmt.Errorf("ids: read random source: %w", err)
	}
	for i, b := range buf {
		buf[i] = alphabet[b&0x1f]
	}
	return string(buf), nil
}

// New mints a fresh id for kind using crypto/rand.
func New(kind Kind) (string, error) { return defaultGen.New(kind) }

// NewUnique mints a collision-free id for kind using crypto/rand.
func NewUnique(kind Kind, exists func(string) bool) (string, error) {
	return defaultGen.NewUnique(kind, exists)
}

// Parse splits an opaque id into its kind and body, rejecting a missing
// separator, an unregistered prefix, or a malformed body.
func Parse(id string) (Kind, string, error) {
	prefix, body, ok := strings.Cut(id, "_")
	if !ok {
		return "", "", fmt.Errorf("ids: %q is missing the '<prefix>_' separator", id)
	}
	kind, ok := prefixKind[prefix]
	if !ok {
		return "", "", fmt.Errorf("ids: %q has an unknown prefix %q", id, prefix)
	}
	if !validBody(body) {
		return "", "", fmt.Errorf("ids: %q has a malformed body", id)
	}
	return kind, body, nil
}

// Valid reports whether id is a well-formed opaque id of any known kind.
func Valid(id string) bool {
	_, _, err := Parse(id)
	return err == nil
}

// ValidFor reports whether id is well-formed and its prefix matches kind.
func ValidFor(kind Kind, id string) bool {
	k, _, err := Parse(id)
	return err == nil && k == kind
}

// validBody reports whether s is exactly bodyLen characters, all drawn from
// the Crockford alphabet.
func validBody(s string) bool {
	if len(s) != bodyLen {
		return false
	}
	for _, c := range s {
		if !strings.ContainsRune(alphabet, c) {
			return false
		}
	}
	return true
}
