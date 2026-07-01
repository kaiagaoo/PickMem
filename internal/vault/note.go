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
)

// Frontmatter is the YAML block at the top of a memory note. Field order in
// serialization is fixed by MarshalYAML so notes on disk stay diff-friendly.
type Frontmatter struct {
	ID             string    `yaml:"id"`
	Label          string    `yaml:"label"`
	Group          string    `yaml:"group"`
	Tags           []string  `yaml:"tags,omitempty"`
	Source         string    `yaml:"source"`
	Status         string    `yaml:"status"`
	CreatedAt      time.Time `yaml:"created_at"`
	SuggestedGroup string    `yaml:"suggested_group,omitempty"`
}

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
	var buf bytes.Buffer
	buf.WriteString("---\n")
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&n.Frontmatter); err != nil {
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
