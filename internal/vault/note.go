package vault

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v3"
)

const (
	SourceManual  = "manual"
	SourceImport  = "import"
	SourceExtract = "extract"

	StatusActive  = "active"
	StatusPending = "pending"

	// Kinds of note. Type is what you're storing, independent of Group
	// (where it lives) — it lets a pick target "my ideas about X" rather
	// than every note under X. TypeFact is the default and is omitted on
	// disk, so fact notes and pre-type notes stay byte-clean; only the
	// non-default kinds carry an explicit `type:` line.
	TypeFact      = "fact"      // a stable fact — the classic "memory"
	TypeIdea      = "idea"      // a proposal or concept to develop
	TypeThought   = "thought"   // a fleeting reflection, not yet resolved
	TypeReference = "reference" // external material: a quote, link, excerpt
)

// NormalizeType maps a raw type string to a known kind, defaulting to
// TypeFact for empty or unrecognized values so a slightly-future note is
// treated as a plain memory rather than dropped.
func NormalizeType(s string) string {
	switch s {
	case TypeIdea, TypeThought, TypeReference:
		return s
	default:
		return TypeFact
	}
}

// Frontmatter is the YAML block at the top of a memory note. Field order in
// serialization is fixed by MarshalYAML so notes on disk stay diff-friendly.
type Frontmatter struct {
	ID             string    `yaml:"id"`
	Label          string    `yaml:"label"`
	Group          string    `yaml:"group"`
	Type           string    `yaml:"type,omitempty"`
	Tags           []string  `yaml:"tags,omitempty"`
	Source         string    `yaml:"source"`
	Status         string    `yaml:"status"`
	CreatedAt      time.Time `yaml:"created_at"`
	SuggestedGroup string    `yaml:"suggested_group,omitempty"`
}

// Kind returns the note's normalized type (TypeFact for the default/empty
// case). Prefer this over reading Type directly when you need a concrete
// kind to display or filter on.
func (f Frontmatter) Kind() string { return NormalizeType(f.Type) }

// Note is a single memory item: its parsed frontmatter, body, and the vault-
// relative path where it lives on disk.
type Note struct {
	Frontmatter
	Body string
	// RelPath is the note's path relative to the vault root, using forward
	// slashes. Empty until the note is written to (or read from) disk.
	RelPath string
}

// ParseNote parses a full markdown-with-frontmatter file. Returns an error
// if the frontmatter block is missing or malformed.
func ParseNote(data []byte) (*Note, error) {
	n := &Note{}
	rest, err := frontmatter.Parse(bytes.NewReader(data), &n.Frontmatter)
	if err != nil {
		return nil, fmt.Errorf("parse frontmatter: %w", err)
	}
	if n.ID == "" {
		return nil, fmt.Errorf("frontmatter missing required field: id")
	}
	if n.Label == "" {
		return nil, fmt.Errorf("frontmatter missing required field: label")
	}
	n.Body = strings.TrimLeft(string(rest), "\n")
	return n, nil
}

// Serialize renders a Note back to disk bytes: `---\n<yaml>---\n\n<body>\n`.
// Uses an explicit yaml.Encoder so field order matches the struct layout.
func (n *Note) Serialize() ([]byte, error) {
	// Canonicalize the default kind to empty so fact notes serialize
	// without a `type:` line (omitempty), keeping them byte-clean.
	fm := n.Frontmatter
	if NormalizeType(fm.Type) == TypeFact {
		fm.Type = ""
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&fm); err != nil {
		return nil, fmt.Errorf("encode frontmatter: %w", err)
	}
	if err := enc.Close(); err != nil {
		return nil, fmt.Errorf("close encoder: %w", err)
	}
	buf.WriteString("---\n\n")
	buf.WriteString(strings.TrimRight(n.Body, "\n"))
	buf.WriteByte('\n')
	return buf.Bytes(), nil
}
