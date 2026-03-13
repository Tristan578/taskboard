# Contributing to Player2 Kanban

First off, thanks for taking the time to contribute! **Player2 Kanban** is an agent-native enhancement of the original [Taskboard](https://github.com/tcarac/taskboard) project.

## Code of Conduct

This project and everyone participating in it is governed by our Code of Conduct. By participating, you are expected to uphold this code.

## How Can I Contribute?

### Reporting Bugs

- Use the GitHub Issue tracker.
- Check if the bug has already been reported.
- If not, create a new issue. Include a clear title and description, as much relevant information as possible, and a code sample or an executable test case demonstrating the expected behavior that is not occurring.

### Suggesting Enhancements

- Use the GitHub Issue tracker.
- Provide a clear and descriptive title.
- Describe the current behavior and the behavior you'd like to see instead.

### Pull Requests

The `main` branch is protected. All contributions must follow this process:

1. Fork the repo and create your branch from `main`.
2. Implement your changes following the **Brick Building** standard (tests for both success and failure paths).
3. Ensure Go statement coverage does not decrease (CI will fail if total coverage drops below 75%).
4. Verify your changes pass all CI checks:
   - **Backend**: Go tests, coverage, and `gosec` security scan.
   - **Frontend**: Linting, type-checking, and build.
5. All conversations and comments on the PR must be **resolved** before merging.
6. Use **Squash and Merge** or **Rebase and Merge** to maintain a linear history. Merge commits are disabled.

## Engineering Standards

- **Tests**: If you've added code, you *must* add tests.
- **Security**: No secrets or credentials should ever be committed.
- **Code Style**: Run `go fmt` and ensure `npm run lint` passes.

## Attribution

Player2 Kanban is based on [tcarac/taskboard](https://github.com/tcarac/taskboard). We are grateful to the original authors for providing such a solid foundation.

## Development Setup

### Backend (Go)
```bash
go mod tidy
go build ./cmd/kanban
```

### Frontend (React)
```bash
cd web
npm install
npm run dev
```

### Running Tests
```bash
go test ./...
```

## License

By contributing, you agree that your contributions will be licensed under its MIT License.
