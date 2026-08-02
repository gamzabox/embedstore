# Go module path migration plan

## Goal

Change the canonical Go module path from `github.com/embedstore/embedstore` to
`github.com/gamzabox/embedstore`, matching the repository owner.  Keep the
package name, CLI binary name, public Go symbols, `.embed` format, and runtime
behaviour unchanged.

## Scope

1. Update the module directive in `go.mod`.
2. Rewrite every in-repository import of the old module path:
   - `cmd/embedstore/build.go`
   - `cmd/embedstore/inspect.go`
   - `cmd/embedstore/main.go`
   - `cmd/embedstore/search.go`
   - `cmd/embedstore/validate.go`
   - `cmd/embedstore/e2e_inspect_verify_test.go`
   - `cmd/embedstore/e2e_reuse_test.go`
   - `cmd/embedstore/main_test.go`
   - `cmd/embedstore/search_test.go`
   The OpenAI subpackage imports in `build.go`, `main.go`, and `search.go` must
   become `github.com/gamzabox/embedstore/embedding/openai`.
3. Replace placeholder module paths in public documentation:
   - `README.md`: `go install`, `go get`, and both API-example imports.
   - `docs/ARCHITECTURE.md`: Go module identifier table entry.
4. Update the historical/current module-path references in `tasks/PROGRESS.md`
   so it records the new canonical path and the completed migration/verification
   result.
5. Inspect `.github/workflows/ci.yml`, `.github/workflows/release.yml`, and
   `.goreleaser.yaml` after the change.  They use repository-relative checkout,
   `go.mod`, `./cmd/embedstore`, and release token conventions, so no module
   path text change is expected; retain them unchanged unless an exact stale
   module-path reference is found.

## Explicit exclusions

- Do not add a `replace` directive or try to support both import paths from one
  module. Go module identity is the `go.mod` path; preserving the old path
  would require a separately published compatibility module/repository and is
  outside this requested migration.
- Do not change the major module version, public symbol names, CLI commands,
  file format version, embedding behaviour, dependencies, CI triggers, or
  GoReleaser target matrix.
- Do not create a Git tag, GitHub Release, commit, or push in this task.

## Compatibility and release impact

- This is a public import-path migration. Consumers must change imports from
  `github.com/embedstore/embedstore` to `github.com/gamzabox/embedstore` and
  update their dependency command. Existing source imports do not remain
  source-compatible without an external forwarding/compatibility module.
- The Go package identifier remains `embedstore`, so only import paths—not
  ordinary package-qualified calls—change.
- Published releases must originate from the `gamzabox/embedstore` repository
  and use the same tag (`v*`) release workflow. No GoReleaser configuration
  currently embeds the old path; confirm this with a repository-wide search.
- The `.embed` format and already-created `.embed` files are unaffected, since
  the module path is not encoded in the format contract.

## Actor implementation order (TDD)

1. Add a compile-time/import regression test or update an existing CLI package
   test to import `github.com/gamzabox/embedstore`; before changing `go.mod`,
   it should demonstrate the intended canonical import is unresolved.
2. Change `go.mod`, then replace all exact old import strings in production and
   test Go files. Run `gofmt` on every changed Go file.
3. Update README, Architecture, and PROGRESS in the same change. Replace all
   `<organization>` and old concrete module-path examples; do not leave either
   in user-facing installation/API text.
4. Run exact-string searches for both the old path and placeholder to prove
   there are no stale references, excluding only `.git`.

## Acceptance criteria and verification

- `go.mod` starts with `module github.com/gamzabox/embedstore`.
- `rg -n -F 'github.com/embedstore/embedstore' . -g '!vendor' -g '!.git/**'`
  returns no matches.
- `rg -n -F 'github.com/<organization>/embedstore' . -g '!vendor' -g '!.git/**'`
  returns no matches.
- All CLI source/tests import the canonical module and OpenAI subpackage path.
- README install/API snippets and Architecture's module table use the canonical
  path; PROGRESS records this work as complete with the validation result.
- `.github/workflows/*` and `.goreleaser.yaml` have been reviewed and retain
  their correct repository-relative configuration; `LICENSE` remains present
  for GoReleaser archives.
- Run `gofmt -w` for changed Go files, `go test ./...`, `go vet ./...`, and
  `git diff --check`.
- Because the module's public API/import contract changes, also run
  `go test -race ./...`. If the known ThreadSanitizer VMA environment failure
  prevents it, report that limitation and rely on the Linux GitHub Actions CI
  race job for supported-environment confirmation.

## Risks to evaluate independently

- A partial replacement can leave CLI tests resolving the old module from a
  module cache instead of the local package; the exact repository search and
  full test run guard against it.
- Placeholder examples are not compilable copy/paste instructions; all
  user-facing occurrences must be converted to the real owner path.
- This changes external Go consumers' import path. The evaluator should ensure
  documentation calls this out or that release notes/PROGRESS make the impact
  discoverable, without misleadingly claiming old imports still work.
