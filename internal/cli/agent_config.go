package cli

import (
	"bytes"
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

//go:embed agent-skills/base-instructions.md
var agentSkillsFS embed.FS

type agentTarget struct {
	path       string
	dir        string // directory to create before writing, empty if none
	preContent string // content prepended before base instructions
}

var agentTargets = map[string]agentTarget{
	"devin": {path: ".devin/player2.md", dir: ".devin"},
	"cursor": {
		path:       ".cursor/rules/player2.mdc",
		dir:        ".cursor/rules",
		preContent: "---\ndescription: Mandatory workflow for Player2 Kanban\nglobs: **/*\nalwaysApply: true\n---\n\n",
	},
	"claude": {
		path: "CLAUDE.md",
	},
	"gemini": {
		path: ".gemini/GEMINI.md",
		dir:  ".gemini",
	},
	"windsurf": {
		path: ".windsurfrules",
	},
	"antigravity": {
		path: ".agent/rules/player2.md",
		dir:  ".agent/rules",
	},
	"copilot": {
		path: ".github/copilot-instructions.md",
		dir:  ".github",
	},
	"codex": {
		path: "AGENTS.md",
	},
}

func agentConfigCommands() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-config",
		Short: "Install agent instructions/rules for specific AI IDEs",
	}

	installCmd := &cobra.Command{
		Use:   "install [agent]",
		Short: "Install instructions for an AI agent IDE",
		Long: `Install Player2 Kanban workflow instructions for a specific AI agent IDE.

Supported agents:
  cursor       → .cursor/rules/player2.mdc
  claude       → CLAUDE.md
  gemini       → .gemini/GEMINI.md
  windsurf     → .windsurfrules
  antigravity  → .agent/rules/player2.md
  copilot      → .github/copilot-instructions.md
  codex        → AGENTS.md
  all          → install all of the above`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agent := args[0]
			baseContent, err := agentSkillsFS.ReadFile("agent-skills/base-instructions.md")
			if err != nil {
				return fmt.Errorf("reading base instructions: %w", err)
			}

			if agent == "all" {
				for name, target := range agentTargets {
					if err := installAgentConfig(name, target, baseContent); err != nil {
						return err
					}
				}
				return nil
			}

			target, ok := agentTargets[agent]
			if !ok {
				return fmt.Errorf("unknown agent: %s\nRun 'player2-kanban agent-config install --help' for supported agents", agent)
			}
			return installAgentConfig(agent, target, baseContent)
		},
	}

	cmd.AddCommand(installCmd)
	return cmd
}

func installAgentConfig(name string, target agentTarget, baseContent []byte) error {
	if target.dir != "" {
		_ = os.MkdirAll(target.dir, 0700)
	}

	content := baseContent

	const begin = "<!-- player2:start -->"
	const end = "<!-- player2:end -->"
	existing, err := os.ReadFile(target.path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	block := append([]byte(begin+"\n"), content...)
	block = append(block, []byte("\n"+end+"\n")...)
	start := bytes.Index(existing, []byte(begin))
	finish := bytes.Index(existing, []byte(end))
	var output []byte
	if start >= 0 && finish >= start {
		output = append(output, existing[:start]...)
		output = append(output, block...)
		output = append(output, existing[finish+len(end):]...)
	} else if start >= 0 || finish >= 0 {
		return fmt.Errorf("incomplete Player2 block in %s; repair markers before installing", target.path)
	} else {
		if len(existing) == 0 {
			output = append(output, []byte(target.preContent)...)
		}
		output = append(output, existing...)
		if len(existing) > 0 {
			output = append(output, []byte("\n\n")...)
		}
		output = append(output, block...)
	}
	if err := os.WriteFile(target.path, output, 0600); err != nil {
		return fmt.Errorf("writing %s config: %w", name, err)
	}

	fmt.Printf("Installed %s instructions to %s.\n", name, target.path)
	return nil
}
