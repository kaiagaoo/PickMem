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

	// Well-known tag names. These are just ordinary tags, but the UI offers
	// them as one-click "suggested" chips and gives them distinct colors.
	// A note used to carry a single `type:` field; that was folded into tags
	// (see ParseNote's legacy migration), so these are plain tag values now.
	TagFact      = "fact"      // a stable fact — the classic "memory"
	TagIdea      = "idea"      // a proposal or concept to develop
	TagThought   = "thought"   // a fleeting reflection, not yet resolved
	TagReference = "reference" // external material: a quote, link, excerpt
)

// DefaultSuggestedTags is the built-in set of quick-pick tag chips a vault
// starts with. Users can customize the list (see Config.SuggestedTags); they
// are ordinary tags, not a required vocabulary.
func DefaultSuggestedTags() []string {
	return []string{TagFact, TagIdea, TagThought, TagReference}
}

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

	// Legacy migration: notes written before types were folded into tags
	// carry a `type:` line. Fold a non-default type into the tag list (the
	// Frontmatter struct no longer has a Type field, so read it via a
	// throwaway). `fact` was the implicit default and adds no information, so
	// it's dropped. The `type:` line simply vanishes the next time the note
	// is saved.
	var legacy struct {
		Type string `yaml:"type"`
	}
	if _, err := frontmatter.Parse(bytes.NewReader(data), &legacy); err == nil {
		if t := strings.TrimSpace(legacy.Type); t != "" && t != TagFact {
			n.Tags = appendUniqueTag(n.Tags, t)
		}
	}
	return n, nil
}

// appendUniqueTag adds tag to the front of tags if not already present,
// preserving the note's own tag order after it.
func appendUniqueTag(tags []string, tag string) []string {
	for _, t := range tags {
		if t == tag {
			return tags
		}
	}
	return append([]string{tag}, tags...)
}

// Serialize renders a Note back to disk bytes: `---\n<yaml>---\n\n<body>\n`.
// Uses an explicit yaml.Encoder so field order matches the struct layout.
func (n *Note) Serialize() ([]byte, error) {
	fm := n.Frontmatter
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
