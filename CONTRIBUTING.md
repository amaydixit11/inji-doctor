# Contributing to inji-doctor

Thank you for your interest in contributing to `inji-doctor`! This tool is designed to help the Inji community diagnose and fix deployment issues quickly.

## How to Contribute

### Adding a New Check

1.  **Define the Check**: Every check must implement the `Checker` interface in `internal/checker/checker.go`.
2.  **Implementation**: Add a new file in `internal/checker/` (e.g., `my_new_check.go`).
3.  **Registration**: Register your check in `internal/checker/registry.go` inside the `allCheckers()` function.
4.  **Test**: Add a corresponding test file (e.g., `my_new_check_test.go`).

### Development Setup

1.  Install Go 1.22 or later.
2.  Clone the repository:
    ```bash
    git clone https://github.com/inji/inji-doctor.git
    cd inji-doctor
    ```
3.  Build the project:
    ```bash
    go build ./...
    ```
4.  Run tests:
    ```bash
    go test ./...
    ```

## Code Style

- Follow standard Go formatting (`go fmt`).
- Write descriptive error messages with clear "Fix" instructions.
- Ensure all logic is covered by unit tests.

## Submitting Pull Requests

1.  Fork the repository.
2.  Create a feature branch.
3.  Commit your changes with clear, descriptive messages.
4.  Push to your fork and submit a PR.
5.  Ensure the CI pipeline passes.

## License

By contributing to this project, you agree that your contributions will be licensed under the Mozilla Public License 2.0.
