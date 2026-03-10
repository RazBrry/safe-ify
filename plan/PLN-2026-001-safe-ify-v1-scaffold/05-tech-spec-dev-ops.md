# PLN-2026-001 Tech Spec Developer: Operations

## 1. Build System

### 1.1 Makefile

```makefile
BINARY_NAME=safe-ify
BUILD_DIR=./bin
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

.PHONY: build clean test lint

build:
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/safe-ify/

clean:
	rm -rf $(BUILD_DIR)

test:
	go test ./... -v -race

test-coverage:
	go test ./... -coverprofile=coverage.out -race
	go tool cover -func=coverage.out

lint:
	golangci-lint run ./...
```

### 1.2 go.mod

```
module github.com/erwinmaasbach/safe-ify

go 1.22

require (
    github.com/spf13/cobra v1.8+
    github.com/charmbracelet/huh v0.6+
    github.com/charmbracelet/lipgloss v1.0+
    gopkg.in/yaml.v3 v3.0+
)
```

Exact versions to be pinned at implementation time.

### 1.3 Distribution

- `go install github.com/erwinmaasbach/safe-ify/cmd/safe-ify@latest`
- `make build` produces `./bin/safe-ify`
- No goreleaser, Homebrew, or container distribution in v1.

---

## 2. Test Strategy

### 2.1 Unit Tests

**Package: `internal/config`**
- `TestLoadGlobal_ValidConfig` -- loads valid YAML.
- `TestLoadGlobal_MissingFile` -- returns ConfigNotFoundError.
- `TestLoadGlobal_InsecurePermissions` -- returns ConfigInsecureError when file is 0644.
- `TestSaveGlobal_CreatesWithCorrectPermissions` -- verifies 0600.
- `TestLoadProject_ValidConfig` -- loads valid project YAML.
- `TestLoadProject_TraversesParents` -- finds config in parent directory.

**Package: `internal/permissions`**
- `TestResolvePermissions_NoRestrictions` -- all commands allowed.
- `TestResolvePermissions_GlobalDeny` -- denied commands removed.
- `TestResolvePermissions_ProjectDeny` -- project denials applied.
- `TestResolvePermissions_ProjectCannotEscalate` -- project cannot re-enable globally denied command.
- `TestResolvePermissions_CombinedDenials` -- both global and project denials merge correctly.
- `TestEnforcer_AllowedCommand` -- returns nil.
- `TestEnforcer_DeniedCommand` -- returns PermissionDeniedError.
- `TestValidateDenyList_ValidCommands` -- accepts valid command names.
- `TestValidateDenyList_InvalidCommand` -- rejects unknown command.

**Package: `internal/audit`**
- `TestLogger_WritesEntry` -- writes formatted log line.
- `TestLogger_AppendsToExisting` -- does not overwrite.

### 2.2 Integration Tests (Mocked Coolify API)

**Package: `internal/coolify`**

Use `net/http/httptest` to create a mock Coolify server.

- `TestClient_Healthcheck_Success` -- mock returns 200.
- `TestClient_Healthcheck_Failure` -- mock returns 401.
- `TestClient_ListApplications` -- mock returns application list JSON.
- `TestClient_Deploy_Success` -- mock returns deployment response.
- `TestClient_Deploy_RateLimit` -- mock returns 429, verify error message includes retry info.
- `TestClient_Restart_Success` -- mock returns restart response.
- `TestClient_GetLogs` -- mock returns log lines.
- `TestClient_GetApplication` -- mock returns application details.
- `TestClient_NetworkError` -- mock server down, verify error handling.

### 2.3 CLI Integration Tests

**Package: `internal/cli`**

Test full command execution with mocked dependencies:
- `TestDeployCommand_JSON` -- verify JSON output structure.
- `TestDeployCommand_PermissionDenied` -- verify error JSON.
- `TestStatusCommand_JSON` -- verify status output.
- `TestListCommand_JSON` -- verify list output.
- `TestDoctorCommand_Output` -- verify markdown output format.

### 2.4 Test Data

Test fixtures in `testdata/` directory:
- `testdata/valid-global-config.yaml`
- `testdata/valid-project-config.yaml`
- `testdata/insecure-global-config.yaml`

---

## 3. Performance Requirements

- Non-network CLI operations (permission check, config load, help text): < 100ms.
- Measured via Go benchmarks in `*_test.go` files.
- Benchmark tests: `BenchmarkConfigLoad`, `BenchmarkPermissionResolve`.

---

## 4. Definition of Ready (DoR) -- per task

- [ ] Task file exists with all sections filled.
- [ ] Read list references only existing or previously-produced files.
- [ ] Acceptance criteria are measurable.
- [ ] Dependent tasks (earlier in slice) are Done.

## 5. Definition of Done (DoD) -- per task

- [ ] All acceptance criteria met.
- [ ] Code compiles with zero warnings (`go vet ./...`).
- [ ] Unit tests pass (`go test ./... -race`).
- [ ] No linting errors (`golangci-lint run`).
- [ ] Reviewer verdict: PASS.
- [ ] Tester verdict: PASS.

## 6. Definition of Done (DoD) -- per slice

- [ ] All tasks in slice are Done.
- [ ] Gate row approved.
- [ ] Branch merged to main.
- [ ] `go build ./...` succeeds on main after merge.
