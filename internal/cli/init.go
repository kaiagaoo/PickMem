package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/qwgao/pickmem/internal/vault"
	"github.com/qwgao/pickmem/templates"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var bare bool
	var force bool

	cmd := &cobra.Command{
		Use:   "init <path>",
		Short: "Scaffold a new PickMem vault at <path>",
		Long:  "Creates the pickmem/ subdirectory with inbox/, config.json, lenses.json, and active.json, and lays down the starter taxonomy (group folders + routing rules + a vault README). Pass --bare for an empty vault instead. Records the vault path in the user config so subsequent commands don't need --vault.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			target, err := filepath.Abs(args[0])
			if err != nil {
				return err
			}
			if err := os.MkdirAll(target, 0o755); err != nil {
				return fmt.Errorf("create vault dir: %w", err)
			}

			existed := hasExistingVault(target)
			if existed && !force {
				fmt.Fprintf(cmd.OutOrStdout(), "Vault already initialized at %s (use --force to reapply the starter taxonomy).\n", target)
			}

			applyTemplate := !bare && (!existed || force)

			// Apply the template BEFORE vault.Init so the template's
			// pickmem/config.json (with routing rules) lands first.
			// vault.Init won't overwrite it — it only writes defaults for
			// files that don't already exist.
			if applyTemplate {
				if err := templates.Apply(templates.DefaultName, target); err != nil {
					return fmt.Errorf("apply starter taxonomy: %w", err)
				}
			}
			s, err := vault.Init(target)
			if err != nil {
				return err
			}
			// Stamp the template name onto whatever config is now on disk
			// (either the template's or the freshly-written default).
			if applyTemplate {
				cfg, err := s.LoadConfig()
				if err != nil {
					return err
				}
				cfg.TemplateName = templates.DefaultName
				if err := s.SaveConfig(cfg); err != nil {
					return err
				}
			}

			// Remember this vault so daily commands don't need --vault.
			if err := SaveUserConfig(UserConfig{VaultPath: target}); err != nil {
				return fmt.Errorf("save user config: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initialized PickMem vault at %s\n", target)
			if applyTemplate {
				fmt.Fprintln(cmd.OutOrStdout(), "Laid down the starter taxonomy — see README.md in the vault for the map.")
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Set as default vault. Try: pickmem add --label ... --group ...")
			return nil
		},
	}
	cmd.Flags().BoolVar(&bare, "bare", false, "skip the starter taxonomy; create an empty vault")
	cmd.Flags().BoolVar(&force, "force", false, "re-apply the starter taxonomy even if the vault is already initialized")
	return cmd
}

// hasExistingVault reports whether the target already looks like a
// PickMem vault (has the pickmem/ subdir).
func hasExistingVault(path string) bool {
	_, err := os.Stat(filepath.Join(path, vault.PickmemDir, vault.ConfigFile))
	return err == nil
}
