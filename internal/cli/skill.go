package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/nortezh/cli/internal/cli/skill"
)

// skillTargets maps target name → user-level skill directory.
//
//   - claude: Claude Code (also read by opencode via its Claude-Code compat layer)
//   - codex:  OpenAI Codex CLI (~/.agents/skills/ — multi-file SKILL.md format)
//
// Both tools use the same `<dir>/SKILL.md` layout, so the embedded SKILL.md
// drops into either location unchanged.
var skillTargets = map[string]func(home string) string{
	"claude": func(home string) string { return filepath.Join(home, ".claude", "skills", "ntzh") },
	"codex":  func(home string) string { return filepath.Join(home, ".agents", "skills", "ntzh") },
}

func newSkillCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skill",
		Short: "Manage the bundled AI-coding-agent skill for ntzh",
		Long: `The ntzh binary ships an embedded SKILL.md that teaches AI
coding agents how to use this CLI (flag shapes, project/location
resolution, common recipes).

Supported targets:
  claude   ~/.claude/skills/ntzh/SKILL.md  (Claude Code; also picked up by opencode)
  codex    ~/.agents/skills/ntzh/SKILL.md  (OpenAI Codex CLI)`,
	}
	cmd.AddCommand(newSkillInstallCmd())
	return cmd
}

func newSkillInstallCmd() *cobra.Command {
	var (
		dir     string
		target  string
		force   bool
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install the bundled skill for Claude Code, Codex, or both",
		Long: `Write the embedded SKILL.md into the per-tool skills directory.

--target accepts 'claude', 'codex', or 'all' (default). With --dir set,
exactly one SKILL.md is written to that directory regardless of --target.

Skips writing if SKILL.md already exists; pass --force to overwrite.`,
		Example: `  ntzh skill install                       # install for both claude and codex
  ntzh skill install --target=claude
  ntzh skill install --target=codex --force
  ntzh skill install --dir=/custom/path    # write to one explicit directory`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if dir != "" {
				return writeSkill(cmd, dir, force)
			}
			targets, err := resolveTargets(target)
			if err != nil {
				return err
			}
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("locate home dir: %w", err)
			}
			for _, name := range targets {
				dest := skillTargets[name](home)
				if err := writeSkill(cmd, dest, force); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&dir, "dir", "", "explicit destination directory; bypasses --target")
	cmd.Flags().StringVar(&target, "target", "all", "install target: claude, codex, or all")
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing SKILL.md")
	return cmd
}

func resolveTargets(target string) ([]string, error) {
	switch target {
	case "", "all":
		return []string{"claude", "codex"}, nil
	}
	if _, ok := skillTargets[target]; !ok {
		valid := make([]string, 0, len(skillTargets)+1)
		for k := range skillTargets {
			valid = append(valid, k)
		}
		valid = append(valid, "all")
		return nil, fmt.Errorf("unknown --target %q (valid: %s)", target, strings.Join(valid, ", "))
	}
	return []string{target}, nil
}

func writeSkill(cmd *cobra.Command, dest string, force bool) error {
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
}
