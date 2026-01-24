# AGENTS.md

Instructions for AI agents working with this repository.

## 🔥 CRITICALLY IMPORTANT: Snapshot tests for UI

### Why this is important

This project uses **snapshot testing** for UI components (TUI). This is critically important for preventing accidental changes to the visual part of the application.

**IMPORTANT:** Any changes to UI code can break the layout. Snapshot tests ensure that the design remains stable.

### What to do BEFORE making UI changes

**MUST** run snapshot tests:

```bash
# Run file tree tests (MANDATORY)
go test -v ./cmd/dive/cli/internal/ui/v2/panes/filetree

# Or run all UI tests at once
go test -v ./cmd/dive/cli/internal/ui/v2/panes/...
```

### What to do AFTER making UI changes

If you changed something in the UI (styles, layout, spacing, etc.):

1. **MUST** update snapshot files:
   ```bash
   task unit-update-snapshots
   ```

2. **VERIFY** that the changes are visually correct:
   ```bash
   go test -v ./cmd/dive/cli/internal/ui/v2/panes/...
   ```

3. **ENSURE** that all tests pass

### Rules for working with UI code

#### ✅ ACCEPTABLE CHANGES:

- Changing component logic (if nothing changes visually)
- Performance optimization
- Refactoring (if the result is visually identical)
- Adding new features (with snapshot update)

#### ❌ PROHIBITED:

- Making UI changes WITHOUT running tests
- Ignoring failing snapshot tests
- Updating snapshot files "just in case" (only if there are actual changes)

### What to do if tests fail

1. **Look at the diff** - the test will show exactly what changed
2. **If the change is EXPECTED** (you intentionally changed the design):
   - Update snapshot: `task unit-update-snapshots`
3. **If the change is UNEXPECTED** (accidentally broke the layout):
   - Fix the code
   - DO NOT update snapshot files

### Snapshot test structure

```
cmd/dive/cli/internal/ui/v2/panes/
├── filetree/
│   ├── pane.go
│   ├── pane_test.go           # Tests
│   └── __snapshots__/
│       └── pane_test.snap     # Golden files (reference)
├── layers/
│   ├── pane.go
│   ├── pane_test.go
│   └── __snapshots__/
│       └── pane_test.snap
├── details/
│   ├── pane.go
│   ├── pane_test.go
│   └── __snapshots__/
│       └── pane_test.snap
└── image/
    ├── pane.go
    ├── pane_test.go
    └── __snapshots__/
        └── pane_test.snap
```

### Additional resources

- Snapshot test documentation: `cmd/dive/cli/internal/ui/v2/panes/README.md`
- [go-snaps documentation](https://github.com/gkampitakis/go-snaps)

---

## Other important instructions

### Running all tests

```bash
# All unit tests
task unit

# All tests (unit + CLI)
task test
```

### Code quality checks

```bash
# Formatting
task format

# Linting
task lint

# All checks
task pr-validations
```

---

## 🚨 CHECKLIST for agents

Before saying "I'm done", the agent MUST:

- [ ] Run `go test -v ./cmd/dive/cli/internal/ui/v2/panes/filetree`
- [ ] If tests fail → understand why (accidental change or real one)
- [ ] If making UI changes → update snapshot files
- [ ] All tests pass SUCCESSFULLY

**DON'T FORGET ABOUT SNAPSHOT TESTS! This is critically important for UI stability!** 🎨
