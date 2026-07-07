package cli

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kaiagaoo/PickMem/internal/mcp"
	"github.com/spf13/cobra"
)

func newContextCmd() *cobra.Command {
	var copyFlag bool

	cmd := &cobra.Command{
		Use:   "context",
		Short: "Print the assembled memory block the model receives",
		Long: `Print the exact context block your current selection assembles — the same
bytes the MCP server and the extension deliver. Useful to verify what the
model sees, and with --copy it becomes a delivery channel of its own:
paste the block into any chat UI, no extension needed.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			s, err := openVault(cmd)
			if err != nil {
				return err
			}
			text, err := mcp.AssembleActive(s)
			if err != nil {
				return err
			}
			if copyFlag {
				if err := copyToClipboard(text); err != nil {
					return fmt.Errorf("copy to clipboard: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Copied to clipboard.")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), text)
			return nil
		},
	}
	cmd.Flags().BoolVar(&copyFlag, "copy", false, "copy the block to the clipboard instead of printing it")
	return cmd
}

// copyToClipboard shells out to the platform clipboard tool. No CGo, no
// extra dependency — the same trade every small CLI makes.
func copyToClipboard(text string) error {
	var candidates [][]string
	switch runtime.GOOS {
	case "darwin":
		candidates = [][]string{{"pbcopy"}}
	case "windows":
		candidates = [][]string{{"clip"}}
	default: // linux and friends: X11, Wayland, or xsel
		candidates = [][]string{{"xclip", "-selection", "clipboard"}, {"wl-copy"}, {"xsel", "--clipboard", "--input"}}
	}
	for _, c := range candidates {
		path, err := exec.LookPath(c[0])
		if err != nil {
			continue
		}
		cmd := exec.Command(path, c[1:]...)
		cmd.Stdin = strings.NewReader(text)
		return cmd.Run()
	}
	return fmt.Errorf("no clipboard tool found (tried %s)", clipboardNames(candidates))
}

func clipboardNames(cs [][]string) string {
	names := make([]string, len(cs))
	for i, c := range cs {
		names[i] = c[0]
	}
	return strings.Join(names, ", ")
}
