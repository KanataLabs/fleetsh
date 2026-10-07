# fleetsh Foundation Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Establish the authorized public Go/GPL repository and gh-pages documentation.

**Architecture:** A minimal CLI calls an isolated configuration initializer.
Main stores documentation source; gh-pages stores the published Jekyll tree.

**Tech Stack:** Go 1.27, standard library, GitHub Actions, GitHub Pages/Jekyll.

---

### Task 1: Executable foundation

Create go.mod, cmd/fleetsh/main.go, internal/cli/cli.go and internal/config/config.go.
Embed internal/config/example.toml. Implement help, version and init only.
Test invalid arguments, explicit paths, private file creation and refusal to
overwrite an existing configuration in internal/cli/cli_test.go and
internal/config/config_test.go. Run gofmt, go test ./..., go vet ./... and go build.

### Task 2: Project and product documentation

Create LICENSE from the authoritative GPL v3 text. Create README.md,
README.zh-CN.md, CONTRIBUTING.md, SECURITY.md and issue/PR templates.
Preserve the full normalized product brief in docs/product.md.
Create docs/{index,getting-started,configuration,architecture,security,roadmap}.md.
State current implementation status and resolve the brief's conflicting priorities.

### Task 3: Automation and publication

Create .github/workflows/ci.yml with OS tests and six CGO-free cross-build targets.
Create docs/_config.yml, docs/_layouts/default.html and docs/assets/style.css.
Create .github/workflows/docs.yml to sync main documentation to gh-pages and request
a Pages build. Seed gh-pages before enabling the workflow and configure Pages
to use gh-pages at repository root.

### Task 4: Verify the concrete result

Create KanataLabs/fleetsh as public, push main and gh-pages, enable private security
reporting. Verify CI, documentation publishing, Pages deployment and the live site.
Record exact limitations without claiming remote SSH functionality exists.
