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
		Short: "Install instructions for a specific agent (cursor, claude, gemini, windsurf, antigravity, copilot)",
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
				targetPath = ".cursorrules"
			case "claude":
				targetPath = "CLAUDE.md" // Official Claude Code standard
			case "gemini":
				targetPath = ".gemini/GEMINI.md"
				os.MkdirAll(".gemini", 0755)
			case "windsurf":
				targetPath = ".windsurfrules"
			case "antigravity":
				targetPath = ".agent/rules/player2.md"
				os.MkdirAll(".agent/rules", 0755)
			case "copilot":
				targetPath = ".github/copilot-instructions.md"
				os.MkdirAll(".github", 0755)
			default:
				return fmt.Errorf("unknown agent: %s. Supported: cursor, claude, gemini, windsurf, antigravity, copilot", agent)
			}

			if err := os.WriteFile(targetPath, baseContent, 0644); err != nil {
				return fmt.Errorf("writing agent config: %w", err)
			}

			fmt.Printf("Installed %s instructions to %s.\n", agent, targetPath)
			return nil
		},
	}

	cmd.AddCommand(installCmd)
	return cmd
}
