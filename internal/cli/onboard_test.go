package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/kaiagaoo/PickMem/internal/vault"
)

func TestRunOnboardAnswersSkipsAndQuit(t *testing.T) {
	s, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// Answer Q1, skip Q2, answer Q3, then quit on Q4.
	in := strings.NewReader("Shanghai, UTC+8\n\nBuilds developer tools in Go\nq\n")
	var out bytes.Buffer

	created, err := runOnboard(s, in, &out)
	if err != nil {
		t.Fatal(err)
	}
	if created != 2 {
		t.Fatalf("created = %d, want 2", created)
	}

	notes := s.ListActive()
	if len(notes) != 2 {
		t.Fatalf("vault has %d active notes, want 2", len(notes))
	}
	byLabel := map[string]*vault.Note{}
	for _, n := range notes {
		byLabel[n.Label] = n
	}
	loc := byLabel[onboardQuestions[0].Label]
	if loc == nil || loc.Group != onboardQuestions[0].Group || loc.Body != "Shanghai, UTC+8" {
		t.Errorf("first answer wrong: %+v", loc)
	}
	role := byLabel[onboardQuestions[2].Label]
	if role == nil || role.Group != onboardQuestions[2].Group {
		t.Errorf("third answer wrong: %+v", role)
	}
	// All onboard notes are active + manual — the user typed them.
	for _, n := range notes {
		if n.Status != vault.StatusActive || n.Source != vault.SourceManual {
			t.Errorf("note %q status=%s source=%s, want active/manual", n.Label, n.Status, n.Source)
		}
	}
	if !strings.Contains(out.String(), "Created 2 memories") {
		t.Errorf("summary missing:\n%s", out.String())
	}
}

func TestRunOnboardEOFKeepsAnswersSoFar(t *testing.T) {
	s, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// One answer, then the stream ends (user hit Ctrl-D / piped input ran dry).
	created, err := runOnboard(s, strings.NewReader("Beijing\n"), &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if created != 1 || len(s.ListActive()) != 1 {
		t.Fatalf("created = %d, active = %d, want 1/1", created, len(s.ListActive()))
	}
}

func TestRunOnboardAllSkipped(t *testing.T) {
	s, err := vault.Init(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	created, err := runOnboard(s, strings.NewReader("q\n"), &out)
	if err != nil {
		t.Fatal(err)
	}
	if created != 0 || len(s.ListActive()) != 0 {
		t.Fatalf("expected empty vault, got created=%d", created)
	}
	if !strings.Contains(out.String(), "No memories created") {
		t.Errorf("empty-run message missing:\n%s", out.String())
	}
}
