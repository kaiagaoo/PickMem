package cli

import (
	"bufio"
	"fmt"
	"io"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
)

// onboardQuestion is one interview prompt. Questions are deliberately
// narrow so that one answer maps cleanly onto one memory note — no
// AI-splitting needed, which keeps onboarding fully local.
type onboardQuestion struct {
	Group  string
	Label  string
	Type   string // vault.TypeFact unless noted
	Prompt string
}

// onboardQuestions is the interview bank, ordered to start easy (identity,
// work) and leave the most personal areas (health, finance) for later,
// once the skip mechanic is familiar. Every question is skippable.
var onboardQuestions = []onboardQuestion{
	{Group: "about/identity", Label: "location", Prompt: "Where are you based (city, timezone)?"},
	{Group: "about/identity", Label: "languages", Prompt: "What languages do you speak, and which do you prefer for answers?"},
	{Group: "work/role", Label: "role", Prompt: "What's your role, and what do you actually do day to day?"},
	{Group: "work/stack", Label: "stack", Prompt: "What's your main tech stack / the tools you work in?"},
	{Group: "work/projects", Label: "current work", Prompt: "What are you working on right now?"},
	{Group: "about/preferences", Label: "tool preferences", Prompt: "Any tools you insist on (or refuse to use)? Editor, OS, workflow…"},
	{Group: "about/preferences", Label: "answer style", Prompt: "How do you like an assistant to answer — concise or detailed? Any pet peeves?"},
	{Group: "projects", Label: "side projects", Prompt: "Any side projects or hobbies you're invested in?"},
	{Group: "learning/topics", Label: "learning", Type: vault.TypeThought, Prompt: "What are you learning, or wanting to learn?"},
	{Group: "relationships/family", Label: "people", Prompt: "People you mention often — partner, kids, close friends? (skip if private)"},
	{Group: "home/logistics", Label: "living situation", Prompt: "Anything about your living situation that matters — commute, household, schedule? (skip if private)"},
	{Group: "about/health", Label: "health", Prompt: "Health facts an assistant should factor in — allergies, conditions, diet? (skip if private)"},
	{Group: "finance/goals", Label: "financial context", Prompt: "Financial goals or constraints you'd want factored into advice? (skip if private)"},
}

func newOnboardCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "onboard",
		Short: "Seed your vault by answering a short interview",
		Long: `Build your first memories through a quick Q&A. Each answer becomes one
active memory note, already filed into a group — the fastest way from an
empty vault to a useful one.

One line per answer. Press Enter to skip a question, type q to stop early
(everything answered so far is kept). Safe to run again later; it only
adds notes, never edits existing ones.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := vaultFlag(cmd)
			if err != nil {
				return err
			}
			s, err := vault.Open(root)
			if err != nil {
				return err
			}
			_, err = runOnboard(s, cmd.InOrStdin(), cmd.OutOrStdout())
			return err
		},
	}
	return cmd
}

// runOnboard drives the interview loop. Split from the cobra wiring so
// tests can feed answers through a reader and inspect the store after.
func runOnboard(s *vault.Store, in io.Reader, out io.Writer) (int, error) {
	fmt.Fprintf(out, "Let's seed your vault — %d quick questions.\n", len(onboardQuestions))
	fmt.Fprintln(out, "One line per answer · Enter to skip · q to stop (answers so far are kept)")

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	created := 0
	groups := map[string]bool{}
	for i, q := range onboardQuestions {
		fmt.Fprintf(out, "\n[%d/%d] %s\n> ", i+1, len(onboardQuestions), q.Prompt)
		if !scanner.Scan() {
			break // EOF — keep what we have
		}
		answer := strings.TrimSpace(scanner.Text())
		if answer == "" {
			continue
		}
		if answer == "q" || answer == "Q" {
			break
		}
		n := &vault.Note{
			Frontmatter: vault.Frontmatter{
				Label: q.Label,
				Group: q.Group,
				Type:  vault.NormalizeType(q.Type),
			},
			Body: answer,
		}
		if _, err := s.Add(n); err != nil {
			return created, fmt.Errorf("saving %q: %w", q.Label, err)
		}
		created++
		groups[q.Group] = true
		fmt.Fprintf(out, "  ✓ saved to %s\n", q.Group)
	}
	if err := scanner.Err(); err != nil {
		return created, err
	}

	fmt.Fprintln(out)
	if created == 0 {
		fmt.Fprintln(out, "No memories created. Run `pickmem onboard` again anytime.")
		return 0, nil
	}
	fmt.Fprintf(out, "Created %d memories across %d groups.\n", created, len(groups))
	fmt.Fprintln(out, "Next: run `pickmem pick` to choose what a session sees,")
	fmt.Fprintln(out, "or open the vault in Obsidian to edit anything you typed.")
	return created, nil
}
