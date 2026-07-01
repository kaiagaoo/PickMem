package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/qwgao/pickmem/internal/install"
	"github.com/spf13/cobra"
)

func newInstallCmd() *cobra.Command {
	var (
		name    string
		binPath string
		dryRun  bool
	)

	cmd := &cobra.Command{
		Use:   "install <client>",
		Short: "Wire `pickmem serve` into a client's MCP config",
		Long: `Write a pickmem MCP server entry into the given client's config file. Merges with any existing config — other MCP servers are preserved.

Supported clients:
  claude-desktop    macOS/Windows/Linux
  cursor            all platforms

For Cline, the config lives inside a VS Code workspace and is easier to edit by hand — see the README for the JSON snippet.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client := args[0]
			cc, err := install.Resolve(client)
			if err != nil {
				return err
			}
			if binPath == "" {
				binPath, err = os.Executable()
				if err != nil {
					return fmt.Errorf("locate pickmem binary (pass --bin): %w", err)
				}
			}
			entry := install.ServerEntry{
				Command: binPath,
				Args:    []string{"serve"},
			}
			// If the caller invoked --vault, forward it so the client
			// picks the same vault the CLI does. Otherwise leave args
			// minimal and rely on $PICKMEM_VAULT / user config.
			if v, _ := cmd.Flags().GetString("vault"); v != "" {
				entry.Args = append(entry.Args, "--vault", v)
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Client:  %s\n", cc.DisplayName)
			fmt.Fprintf(out, "Config:  %s\n", cc.Path)
			fmt.Fprintf(out, "Entry:   %s %s\n", entry.Command, strings.Join(entry.Args, " "))
			if dryRun {
				fmt.Fprintln(out, "\n(dry run — nothing written)")
				return nil
			}
			if err := install.Install(cc, name, entry); err != nil {
				return err
			}
			fmt.Fprintf(out, "\nInstalled. Restart %s to pick up the change.\n", cc.DisplayName)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "pickmem", "entry name under mcpServers")
	cmd.Flags().StringVar(&binPath, "bin", "", "path to the pickmem binary (defaults to the running binary)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would be written without touching the config file")
	return cmd
}

func newUninstallCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:   "uninstall <client>",
		Short: "Remove the pickmem entry from a client's MCP config",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cc, err := install.Resolve(args[0])
			if err != nil {
				return err
			}
			if err := install.Uninstall(cc, name); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Removed %q from %s (%s).\n", name, cc.DisplayName, cc.Path)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "pickmem", "entry name under mcpServers")
	return cmd
}
