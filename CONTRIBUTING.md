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

1. Fork the repo and create your branch from `main`.
2. If you've added code that should be tested, add tests!
3. If you've changed APIs, update the documentation.
4. Ensure the test suite passes.
5. Make sure your code lints.

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
