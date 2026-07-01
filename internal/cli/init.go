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
	var templateName string
	var force bool

	cmd := &cobra.Command{
		Use:   "init <path>",
		Short: "Scaffold a new PickMem vault at <path>",
		Long:  "Creates the pickmem/ subdirectory with inbox/, config.json, lenses.json, and active.json. Optionally copies a starter taxonomy (--template). Records the vault path in the user config so subsequent commands don't need --vault.",
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
				fmt.Fprintf(cmd.OutOrStdout(), "Vault already initialized at %s (use --force to reapply template).\n", target)
			}

			s, err := vault.Init(target)
			if err != nil {
				return err
			}

			if templateName != "" && (!existed || force) {
				if err := templates.Apply(templateName, target); err != nil {
					return fmt.Errorf("apply template %q: %w", templateName, err)
				}
				cfg, err := s.LoadConfig()
				if err != nil {
					return err
				}
				cfg.TemplateName = templateName
				if err := s.SaveConfig(cfg); err != nil {
					return err
				}
			}

			// Remember this vault so daily commands don't need --vault.
			if err := SaveUserConfig(UserConfig{VaultPath: target}); err != nil {
				return fmt.Errorf("save user config: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Initialized PickMem vault at %s\n", target)
			if templateName != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Applied template: %s\n", templateName)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Set as default vault. Try: pickmem add --label ... --group ...")
			return nil
		},
	}
	cmd.Flags().StringVarP(&templateName, "template", "t", "", "starter taxonomy template (personal|developer|researcher)")
	cmd.Flags().BoolVar(&force, "force", false, "re-apply template even if the vault is already initialized")
	return cmd
}

// hasExistingVault reports whether the target already looks like a
// PickMem vault (has the pickmem/ subdir).
func hasExistingVault(path string) bool {
	_, err := os.Stat(filepath.Join(path, vault.PickmemDir, vault.ConfigFile))
	return err == nil
}
