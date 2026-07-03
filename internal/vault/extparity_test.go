package vault

import (
	"strings"
	"testing"
)

// The Chrome extension writes new notes with its own TypeScript serializer
// (extension/src/vault/writenote.ts). This test pins the exact on-disk
// shape that serializer produces and proves the Go store reads it back
// correctly — the cross-language contract for extension-created notes. If
// this breaks, the two note writers have drifted.
func TestParseNoteAcceptsExtensionSerializedFormat(t *testing.T) {
	// A representative note as serializeNote() emits it: bare scalars,
	// a block-style tags list, an ISO-8601 millisecond timestamp, then a
	// blank line before the body.
	raw := "---\n" +
		"id: 01KWG427SJXFMZG9M23Z2DYWR9\n" +
		"label: income — freelance + salary\n" +
		"group: finance/income\n" +
		"tags:\n" +
		"  - money\n" +
		"  - recurring\n" +
		"source: manual\n" +
		"status: active\n" +
		"created_at: 2026-07-02T00:36:06.578Z\n" +
		"---\n\n" +
		"Freelance ~= $5k/mo, salary $8k/mo.\n"

	n, err := ParseNote([]byte(raw))
	if err != nil {
		t.Fatalf("Go store failed to parse extension-written note: %v", err)
	}
	if !ValidID(n.ID) {
		t.Errorf("extension id %q is not a valid ULID", n.ID)
	}
	if n.Label != "income — freelance + salary" {
		t.Errorf("label = %q", n.Label)
	}
	if n.Group != "finance/income" {
		t.Errorf("group = %q", n.Group)
	}
	if len(n.Tags) != 2 || n.Tags[0] != "money" || n.Tags[1] != "recurring" {
		t.Errorf("tags = %v", n.Tags)
	}
	if n.Source != SourceManual || n.Status != StatusActive {
		t.Errorf("source/status = %s/%s", n.Source, n.Status)
	}
	if n.CreatedAt.IsZero() {
		t.Error("created_at did not parse into a time")
	}
	if !strings.Contains(n.Body, "Freelance") {
		t.Errorf("body = %q", n.Body)
	}
}

// A double-quoted label (what the extension emits when a value contains
// ":" or "#") must also round-trip.
func TestParseNoteAcceptsExtensionQuotedLabel(t *testing.T) {
	raw := "---\n" +
		"id: 01KWG427SJXFMZG9M23Z2DYWR9\n" +
		"label: \"budget: 2026 plan # draft\"\n" +
		"group: finance\n" +
		"source: manual\n" +
		"status: active\n" +
		"created_at: 2026-07-02T00:36:06.578Z\n" +
		"---\n\n" +
		"body\n"
	n, err := ParseNote([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if n.Label != "budget: 2026 plan # draft" {
		t.Errorf("quoted label mis-parsed: %q", n.Label)
	}
}
