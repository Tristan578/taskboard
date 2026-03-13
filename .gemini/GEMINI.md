# Player2 Kanban — Agent Mandates

This document serves as the foundational authority for all AI agents interacting with this repository. These mandates take absolute precedence over general defaults.

## Engineering Standards

- **Brick Building**: Every feature, handler, or method must have corresponding tests covering both positive (happy path) and negative (error/edge case) scenarios.
- **100% Target Coverage**: Aim for 100% statement coverage in Go. CI is currently gated at **75%** as a hard floor, but PRs should not decrease total coverage.
- **Titanium Schema**: Persistence to GitHub must use the hidden HTML comment format: `<!-- player2-metadata:<base64-json> -->`.
- **Deterministic Ordering**: Use LexoRank for ticket positioning.

## Continuous Integration & PR Workflow

The following rules are hard-enforced via GitHub Branch Protection on the `main` branch:

1. **PR Required**: Direct pushes to `main` are strictly forbidden. All changes must arrive via a Pull Request.
2. **Green CI is Mandatory**: No PR can be merged unless all status checks pass:
   - `build-and-test`: Go build, full test suite, coverage check, and security scan (`gosec`).
   - `frontend-checks`: React build, linting, and type-checking.
3. **Linear History Only**: Merge commits are disabled. PRs must be merged via **Squash and Merge** or **Rebase and Merge** to maintain a clean, linear git history.
4. **Resolved Conversations**: All PR comments and discussions must be marked as **Resolved** before merging is allowed.

## AI Behavioral Guidelines

- **Self-Correcting**: If a tool call fails or a build errors, diagnose the root cause and backtrack before re-attempting.
- **Context Efficient**: Combine tool calls where possible. Use sub-agents for turn-intensive research or batch refactoring.
- **No Summary Fatigue**: Do not provide verbose summaries of changes unless explicitly requested. High-signal technical intent only.
