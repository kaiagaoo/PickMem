package ingest

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Format is the wire shape of an import source.
type Format string

const (
	FormatAuto       Format = "auto"       // sniff JSON → bullets → paragraphs
	FormatJSON       Format = "json"       // JSON array of strings or {memory: string}
	FormatBullets    Format = "bullets"    // one bullet per line: -, *, +, or numbered
	FormatParagraphs Format = "paragraphs" // separated by blank lines
)

// Aliases match provider-name-based --format flags to the underlying
// shape. All three exports (ChatGPT, Claude, generic) fall into one of
// the base three, so provider names are just hints.
var formatAliases = map[Format]Format{
	"chatgpt": FormatAuto,
	"claude":  FormatAuto,
	"list":    FormatAuto,
}

// ResolveFormat maps a user-supplied flag to a base Format. Unknown
// formats collapse to Auto rather than error — imports should be
// forgiving.
func ResolveFormat(f string) Format {
	fmt := Format(strings.ToLower(strings.TrimSpace(f)))
	if fmt == "" {
		return FormatAuto
	}
	if canon, ok := formatAliases[fmt]; ok {
		return canon
	}
	switch fmt {
	case FormatJSON, FormatBullets, FormatParagraphs, FormatAuto:
		return fmt
	}
	return FormatAuto
}

// Parse extracts memory-candidate bodies from raw import content. Each
// returned string is one candidate — later stages hash them, derive
// labels, route them to groups, and stage as pending inbox notes.
//
// Empty candidates and anything under MinLen chars are dropped inside
// the specific parsers so callers get a clean list.
func Parse(data []byte, f Format) []string {
	if f == "" {
		f = FormatAuto
	}
	switch f {
	case FormatJSON:
		return parseJSON(data)
	case FormatBullets:
		return parseBullets(string(data))
	case FormatParagraphs:
		return SplitParagraphs(string(data))
	case FormatAuto:
		return parseAuto(data)
	default:
		return parseAuto(data)
	}
}

// parseAuto tries JSON first, then bullets, then paragraphs. Each fall-
// through preserves the "no items found" case so a user pointing at the
// wrong file still gets a clear empty result rather than a wall of noise.
func parseAuto(data []byte) []string {
	trimmed := strings.TrimSpace(string(data))
	if strings.HasPrefix(trimmed, "[") || strings.HasPrefix(trimmed, "{") {
		if out := parseJSON(data); len(out) > 0 {
			return out
		}
	}
	// Bullets are recognizable: majority of non-empty lines start with a
	// bullet marker.
	if looksLikeBullets(trimmed) {
		return parseBullets(trimmed)
	}
	return SplitParagraphs(trimmed)
}

// ---------- JSON ----------

// parseJSON accepts a few shapes people ship memories in:
//   - ["memory 1", "memory 2"]                 (bare array of strings)
//   - [{"memory": "..."}, ...]                  (ChatGPT-ish)
//   - [{"text": "..."}, ...]                    (generic)
//   - [{"content": "..."}, ...]                 (another common shape)
//   - {"memories": [...]}                       (top-level object wrap)
func parseJSON(data []byte) []string {
	// Try top-level object with {"memories": [...]} first.
	var wrapper struct {
		Memories json.RawMessage `json:"memories"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Memories) > 0 {
		return parseJSONArray(wrapper.Memories)
	}
	return parseJSONArray(data)
}

func parseJSONArray(raw json.RawMessage) []string {
	// Bare strings first.
	var strs []string
	if err := json.Unmarshal(raw, &strs); err == nil {
		return filterCandidates(strs)
	}
	// Then objects with a body-ish field.
	var objs []map[string]any
	if err := json.Unmarshal(raw, &objs); err != nil {
		return nil
	}
	out := make([]string, 0, len(objs))
	for _, o := range objs {
		if v := stringField(o, "memory", "text", "content", "body"); v != "" {
			out = append(out, v)
		}
	}
	return filterCandidates(out)
}

func stringField(o map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := o[k].(string); ok && strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ---------- bullets ----------

// bulletRE matches leading markers we treat as list items:
//
//   - item
//   - item
//   - item
//     1. item
//     1) item
var bulletRE = regexp.MustCompile(`^\s*(?:[-*+]|\d+[.)])\s+(.+)$`)

func parseBullets(s string) []string {
	lines := strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
	var out []string
	var cur strings.Builder

	flush := func() {
		txt := strings.TrimSpace(cur.String())
		cur.Reset()
		if len(txt) >= MinLen {
			out = append(out, txt)
		}
	}

	for _, line := range lines {
		if m := bulletRE.FindStringSubmatch(line); m != nil {
			flush()
			cur.WriteString(m[1])
			continue
		}
		if strings.TrimSpace(line) == "" {
			// Blank line ends the current bullet's continuation.
			flush()
			continue
		}
		// Continuation of the previous bullet — indented or wrapped text.
		if cur.Len() > 0 {
			cur.WriteByte(' ')
			cur.WriteString(strings.TrimSpace(line))
		}
	}
	flush()
	return out
}

// looksLikeBullets returns true if a majority of non-empty lines start
// with a bullet marker. Threshold picked to be robust to a scattering
// of section headers or footer lines in the export.
func looksLikeBullets(s string) bool {
	lines := strings.Split(s, "\n")
	total, bulleted := 0, 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		total++
		if bulletRE.MatchString(l) {
			bulleted++
		}
	}
	if total < 3 {
		return false
	}
	return bulleted*2 >= total // >=50%
}

// ---------- shared ----------

// filterCandidates drops empties / too-short entries and trims each.
// Applied uniformly to every parser's output.
func filterCandidates(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		s = strings.TrimSpace(s)
		if len(s) < MinLen {
			continue
		}
		out = append(out, s)
	}
	return out
}
