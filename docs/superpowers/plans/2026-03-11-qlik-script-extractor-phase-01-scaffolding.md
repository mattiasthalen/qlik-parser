# QlikView Script Extractor — Phase 01: Project Scaffolding Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Initialize the Go module, project layout, Makefile, and cobra root command so the binary compiles and `--help` / `version` work.

**Architecture:** Standard Go CLI layout — `main.go` delegates to `cmd/root.go` which sets up cobra. A `cmd/version.go` command reports version. No business logic yet.

**Tech Stack:** Go 1.24, github.com/spf13/cobra, github.com/spf13/viper, github.com/rs/zerolog, Make

**Spec:** `docs/superpowers/specs/2026-03-11-qlik-script-extractor-design.md`

**Parallelism note:** Tasks 1–3 are sequential (module must exist before adding deps). Task 4 depends on Task 3. Tasks 5–6 are independent of each other but depend on Task 4.

---

## Chunk 1: Module and Dependencies

### Task 1: Initialize Go Module

**Files:**
- Create: `go.mod`
- Create: `go.sum` (generated)

- [ ] **Step 1: Initialize the module**

> **Note on module path:** Replace `github.com/your-org/qlik-script-extractor` throughout this plan with the actual GitHub org/user (e.g. `github.com/acme/qlik-script-extractor`). The module path must be consistent in `go.mod`, all `import` statements, and all `go get` commands.

```bash
go mod init github.com/your-org/qlik-script-extractor
```

Expected: `go.mod` created with `module github.com/your-org/qlik-script-extractor` and `go 1.24`

- [ ] **Step 2: Verify go.mod content**

Run: `cat go.mod`
Expected: module line + go version line, no other content yet.

- [ ] **Step 3: Commit**

```bash
git add go.mod
git commit -m "chore: initialize Go module"
```

---

### Task 2: Add Dependencies

**Files:**
- Modify: `go.mod`
- Create: `go.sum`

- [ ] **Step 1: Add all project dependencies**

```bash
go get github.com/spf13/cobra@latest
go get github.com/spf13/viper@latest
go get github.com/rs/zerolog@latest
go get github.com/charmbracelet/bubbletea@latest
go get github.com/charmbracelet/lipgloss@latest
```

- [ ] **Step 2: Tidy the module**

```bash
go mod tidy
```

Expected: No errors. `go.sum` created/updated.

- [ ] **Step 3: Verify dependencies are recorded**

Run: `grep -E "cobra|viper|zerolog|bubbletea|lipgloss" go.mod`
Expected: All five packages appear in `require` block.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add project dependencies"
```

---

### Task 3: Create Makefile

**Files:**
- Create: `Makefile`

- [ ] **Step 1: Write the Makefile**

```makefile
.PHONY: build test lint clean

BINARY := qlik-script-extractor

build:
	go build -o $(BINARY) .

test:
	go test ./... -v -count=1

cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY) coverage.out
```

- [ ] **Step 2: Verify syntax**

Run: `make --dry-run build`
Expected: Prints `go build -o qlik-script-extractor .` with no errors.

> **Note:** `make lint` requires `golangci-lint` to be installed. In the devcontainer it is pre-installed via the `ghcr.io/guiyomh/features/golangci-lint:0` feature. Outside the devcontainer, install it with: `curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin`

- [ ] **Step 3: Commit**

```bash
git add Makefile
git commit -m "chore: add Makefile with build/test/lint/cover targets"
```

---

## Chunk 2: CLI Skeleton

### Task 4: Write root command test first

**Files:**
- Create: `cmd/root_test.go`

- [ ] **Step 1: Write failing test for root command**

```go
package cmd_test

import (
	"bytes"
	"testing"

	"github.com/your-org/qlik-script-extractor/cmd"
)

func TestRootHelp(t *testing.T) {
	buf := &bytes.Buffer{}
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"--help"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("qlik-script-extractor")) {
		t.Errorf("expected help output to contain binary name, got: %s", buf.String())
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run TestRootHelp -v`
Expected: Build error — `cannot find package "github.com/your-org/qlik-script-extractor/cmd"` (the package does not exist yet).

---

### Task 5: Implement root command

**Files:**
- Create: `cmd/root.go`

- [ ] **Step 1: Write root command**

```go
package cmd

import (
	"github.com/rs/zerolog"
	"github.com/spf13/cobra"
)

// NewRootCmd constructs the root cobra command.
func NewRootCmd() *cobra.Command {
	var logLevel string

	root := &cobra.Command{
		Use:   "qlik-script-extractor",
		Short: "Extract QlikView load scripts from .qvw files",
		Long: `qlik-script-extractor recursively scans a directory for QVW files
and extracts the embedded load scripts to .qvs text files.`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			level, err := zerolog.ParseLevel(logLevel)
			if err != nil {
				level = zerolog.Disabled
			}
			zerolog.SetGlobalLevel(level)
			return nil
		},
	}

	root.PersistentFlags().StringVar(&logLevel, "log-level", "disabled",
		"Log level: debug, info, warn, error, disabled")

	return root
}
```

- [ ] **Step 2: Run root test to verify it passes**

Run: `go test ./cmd/... -run TestRootHelp -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add cmd/root.go cmd/root_test.go
git commit -m "feat: add cobra root command with --log-level flag"
```

---

### Task 6: Write version command test first

**Files:**
- Create: `cmd/version_test.go`

- [ ] **Step 1: Write failing test**

```go
package cmd_test

import (
	"bytes"
	"testing"

	"github.com/your-org/qlik-script-extractor/cmd"
)

func TestVersionCmd(t *testing.T) {
	buf := &bytes.Buffer{}
	root := cmd.NewRootCmd()
	root.SetOut(buf)
	root.SetArgs([]string{"version"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if !bytes.Contains([]byte(out), []byte("qlik-script-extractor")) {
		t.Errorf("version output missing binary name, got: %s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/... -run TestVersionCmd -v`
Expected: FAIL — no `version` subcommand registered.

---

### Task 7: Implement version command

**Files:**
- Create: `cmd/version.go`

- [ ] **Step 1: Write version command**

```go
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version is set at build time via -ldflags.
var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "qlik-script-extractor %s\n", Version)
		},
	}
}
```

- [ ] **Step 2: Register version command in NewRootCmd**

Edit `cmd/root.go` — add the following line inside `NewRootCmd()`, after the `root` variable is declared:

```go
root.AddCommand(newVersionCmd())
```

- [ ] **Step 3: Run version test to verify it passes**

Run: `go test ./cmd/... -run TestVersionCmd -v`
Expected: PASS

- [ ] **Step 4: Run all cmd tests**

Run: `go test ./cmd/... -v`
Expected: All PASS

- [ ] **Step 5: Commit**

```bash
git add cmd/version.go cmd/version_test.go cmd/root.go
git commit -m "feat: add version subcommand"
```

---

### Task 8: Create main.go and verify binary builds

**Files:**
- Create: `main.go`

- [ ] **Step 1: Write main.go**

```go
package main

import (
	"os"

	"github.com/your-org/qlik-script-extractor/cmd"
)

func main() {
	root := cmd.NewRootCmd()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Build the binary**

Run: `make build`
Expected: Binary `qlik-script-extractor` created. No compile errors.

- [ ] **Step 3: Smoke test the binary**

Run: `./qlik-script-extractor --help`
Expected: Help text printed, includes "Extract QlikView load scripts".

Run: `./qlik-script-extractor version`
Expected: `qlik-script-extractor dev`

- [ ] **Step 4: Run full test suite**

Run: `make test`
Expected: All tests PASS.

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat: add main.go entrypoint"
```
