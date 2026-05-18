package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/cli/skill"
)

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage the bundled Claude Code skill for ntzh",
		Long: `The ntzh binary ships an embedded Claude Code skill that teaches
Claude how to use this CLI (flag shapes, project/location resolution,
common recipes). Install it once and every project's Claude Code
session will pick it up.`,
	}
	cmd.AddCommand(newSkillInstallCmd())
	return cmd
}

func newSkillInstallCmd() *cobra.Command {
	var (
		dir   string
		force bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the bundled skill into ~/.claude/skills/ntzh/",
		Long: `Write the embedded SKILL.md to ~/.claude/skills/ntzh/SKILL.md
(override the destination with --dir). Skips the write if the file
already exists; pass --force to overwrite.`,
		Example: `  ntzh skill install
  ntzh skill install --force
  ntzh skill install --dir=/custom/path`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dest, err := resolveSkillDir(dir)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return fmt.Errorf("create skill dir: %w", err)
			}
			path := filepath.Join(dest, skill.FileName)
			if !force {
				if _, err := os.Stat(path); err == nil {
					fmt.Fprintf(cmd.OutOrStdout(), "skill already installed at %s (use --force to overwrite)\n", path)
					return nil
				}
			}
			if err := os.WriteFile(path, []byte(skill.Content), 0o644); err != nil {
				return fmt.Errorf("write skill: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "installed skill at %s\n", path)
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "destination directory (default ~/.claude/skills/ntzh)")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing SKILL.md")
	return cmd
}

func resolveSkillDir(dir string) (string, error) {
	if dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home dir: %w", err)
	}
	return filepath.Join(home, ".claude", "skills", "ntzh"), nil
}
