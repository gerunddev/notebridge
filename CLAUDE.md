# Claude Development Guidelines

This document outlines best practices and conventions for working on the notebridge project.

## Go Code Quality Standards

### Linting and Formatting
When any Go file is modified, always run the following commands and fix any issues they report:

```bash
golangci-lint run
go vet ./...
go fmt ./...
```

**Requirements:**
- All `golangci-lint` issues must be resolved before considering the work complete
- All `go vet` warnings must be addressed
- All code must be formatted with `go fmt`

### Error Handling
**Never silently ignore errors.** All errors must be at least logged, even if no other action is taken.

**Good:**
```go
if err := someOperation(); err != nil {
    log.Warn("operation failed", "error", err)
}
```

**Good (cleanup operations):**
```go
defer func() {
    if err := f.Close(); err != nil {
        fmt.Fprintf(os.Stderr, "Warning: failed to close file: %v\n", err)
    }
}()
```

**Bad:**
```go
_ = someOperation() // Silent error ignoring
```

**Exception:** When using `//nolint:errcheck`, always include a comment explaining why:
```go
_ = os.Remove(tmpFile) //nolint:errcheck // cleanup - file may not exist
```

### Testing
- Run tests after making changes: `go test ./...`
- Ensure all tests pass before completing work
- Test file setup operations should use `t.Fatalf()` for error handling
- Test cleanup operations can use `//nolint:errcheck` with explanatory comments

## Build Verification
Always verify the build succeeds:
```bash
go build ./...
```

## Summary Checklist
When modifying Go code:
- [ ] Run `golangci-lint run` and fix all issues
- [ ] Run `go vet ./...` and address all warnings
- [ ] Run `go fmt ./...` to format code
- [ ] Verify all errors are logged (minimum requirement)
- [ ] Run `go test ./...` and ensure all tests pass
- [ ] Run `go build ./...` to verify compilation
