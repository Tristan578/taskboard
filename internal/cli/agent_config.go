package cli

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed agent-skills/base-instructions.md
var agentSkillsFS embed.FS

func agentConfigCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-config",
		Short: "Install agent instructions/rules for specific AI IDEs",
	}

	installCmd := &cobra.Command{
		Use:   "install [agent_name]",
		Short: "Install instructions for a specific agent (cursor, claude, gemini, windsurf, antigravity, copilot, codex)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]
			baseContent, err := agentSkillsFS.ReadFile("agent-skills/base-instructions.md")
			if err != nil {
				return fmt.Errorf("reading base instructions: %w", err)
			}

			var targetPath string
			switch agent {
			case "cursor":
				targetPath = ".cursor/rules/player2.mdc"
				_ = os.MkdirAll(".cursor/rules", 0700)
				// Add Cursor-specific frontmatter
				baseContent = append([]byte("---\ndescription: Mandatory workflow for Player2 Kanban\nglobs: **/*\nalwaysApply: true\n---\n\n"), baseContent...)
			case "claude":
				targetPath = "CLAUDE.md" // Official Claude Code standard
			case "gemini":
				targetPath = ".gemini/GEMINI.md"
				_ = os.MkdirAll(".gemini", 0700)
			case "windsurf":
				targetPath = ".windsurfrules"
			case "antigravity":
				targetPath = ".agent/rules/player2.md"
				_ = os.MkdirAll(".agent/rules", 0700)
			case "copilot":
				targetPath = ".github/copilot-instructions.md"
				_ = os.MkdirAll(".github", 0700)
			case "codex":
				targetPath = "AGENTS.md" // OpenAI Codex standard
			default:
				return fmt.Errorf("unknown agent: %s. Supported: cursor, claude, gemini, windsurf, antigravity, copilot, codex", agent)
			}

			// #nosec G306
			if err := os.WriteFile(targetPath, baseContent, 0600); err != nil {
				return fmt.Errorf("writing agent config: %w", err)
			}

			fmt.Printf("Installed %s instructions to %s.\n", agent, targetPath)
			return nil
		},
	}

	cmd.AddCommand(installCmd)
	return cmd
}
