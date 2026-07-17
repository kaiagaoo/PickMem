package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func newAddCmd() *cobra.Command {
	var (
		label    string
		group    string
		tagsCSV  string
		body     string
		bodyFile string
		toInbox  bool
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Create a new memory note",
		Long: `Create a new memory note. Body comes from --body, --file, stdin (if piped), or $EDITOR if attached to a terminal.

Examples:
  pickmem add --label "salary" --group financial --body "monthly base $8k"
  pickmem add --label "kickoff notes" --group work/Client-Acme --file notes.txt
  echo "grocery list" | pickmem add --label "groceries" --group personal
  pickmem add --label "meeting notes" --group work    # opens $EDITOR`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if label == "" {
				return errors.New("--label is required")
			}
			if group == "" && !toInbox {
				return errors.New("--group is required (or use --inbox)")
			}

			s, err := openVault(cmd)
			if err != nil {
				return err
			}

			resolved, err := resolveBody(cmd.InOrStdin(), body, bodyFile, label)
			if err != nil {
				return err
			}

			n := &vault.Note{
				Frontmatter: vault.Frontmatter{
					Label: label,
					Tags:  splitCSV(tagsCSV),
				},
				Body: resolved,
			}
			if toInbox {
				n.SuggestedGroup = group
				out, err := s.AddInbox(n)
				if err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Staged in inbox: %s (%s)\n", out.ID, out.RelPath)
				return nil
			}
			n.Group = group
			out, err := s.Add(n)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added: %s (%s)\n", out.ID, out.RelPath)
			return nil
		},
	}
	cmd.Flags().StringVarP(&label, "label", "l", "", "short title shown in the picker (required)")
	cmd.Flags().StringVarP(&group, "group", "g", "", "group path, e.g. `financial` or `work/Client-Acme`")
	cmd.Flags().StringVar(&tagsCSV, "tags", "", "comma-separated tag list (e.g. `idea,q3`)")
	cmd.Flags().StringVarP(&body, "body", "b", "", "note body (inline)")
	cmd.Flags().StringVarP(&bodyFile, "file", "f", "", "read note body from this file (`-` for stdin)")
	cmd.Flags().BoolVar(&toInbox, "inbox", false, "stage as pending in the inbox instead of adding directly")
	return cmd
}

// resolveBody figures out where the note body comes from. Precedence:
//  1. --body flag
//  2. --file flag (- means stdin)
//  3. stdin, if piped
//  4. $EDITOR, if attached to a terminal
func resolveBody(stdin io.Reader, body, bodyFile, label string) (string, error) {
	if body != "" {
		return body, nil
	}
	if bodyFile != "" {
		if bodyFile == "-" {
			data, err := io.ReadAll(stdin)
			if err != nil {
				return "", err
			}
			return string(data), nil
		}
		data, err := os.ReadFile(bodyFile)
		if err != nil {
			return "", err
		}
		return string(data), nil
	}
	// If stdin is a pipe (not a TTY), read it.
	if f, ok := stdin.(*os.File); ok && f != nil {
		if !term.IsTerminal(int(f.Fd())) {
			data, err := io.ReadAll(stdin)
			if err != nil {
				return "", err
			}
			if s := strings.TrimSpace(string(data)); s != "" {
				return string(data), nil
			}
		}
	}
	// Otherwise, open $EDITOR with a seeded template.
	return runEditor(label)
}

func runEditor(label string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	tmp, err := os.CreateTemp("", "pickmem-*.md")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	seed := fmt.Sprintf("# %s\n\n<!-- write the body of this memory note. save + quit to add it. -->\n", label)
	if _, err := tmp.WriteString(seed); err != nil {
		_ = tmp.Close()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		return "", err
	}
	if err := launch(editor, tmp.Name()); err != nil {
		return "", fmt.Errorf("editor: %w", err)
	}
	data, err := os.ReadFile(tmp.Name())
	if err != nil {
		return "", err
	}
	cleaned := stripEditorComments(string(data))
	if strings.TrimSpace(cleaned) == "" {
		return "", errors.New("empty body — nothing added")
	}
	return cleaned, nil
}

// stripEditorComments removes the HTML-style helper comments we seed into
// the buffer so they don't get saved into the note.
func stripEditorComments(s string) string {
	var out strings.Builder
	for _, line := range strings.Split(s, "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "<!--") && strings.HasSuffix(t, "-->") {
			continue
		}
		out.WriteString(line)
		out.WriteByte('\n')
	}
	return strings.TrimSpace(out.String()) + "\n"
}

// launch runs an editor on the given file, wiring stdin/stdout/stderr so
// vim/nano/etc. get a proper TTY.
func launch(editor, path string) error {
	// Support editors invoked with args, e.g. EDITOR="code --wait".
	parts := strings.Fields(editor)
	if len(parts) == 0 {
		return errors.New("empty EDITOR")
	}
	name, err := exec.LookPath(parts[0])
	if err != nil {
		return err
	}
	args := append(parts[1:], path)
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// splitCSV parses a comma-separated tag list, trimming whitespace and
// skipping empty entries.
func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if t := strings.TrimSpace(p); t != "" {
			out = append(out, t)
		}
	}
	return out
}
