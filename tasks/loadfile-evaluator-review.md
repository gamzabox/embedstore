# LoadFile options evaluator review

## Scope reviewed

Reviewed `tasks/loadfile-planner-plan.md`, the current public loader in
`file.go`, `load_options_test.go`, and the Go API documentation in README and
the design/requirements documents. This review does not modify Go source.

## Result: blocked

The implementation has the right core shape, but the public-documentation
contract and several required regression cases are not complete. Do not mark
the LoadFile-options rotation complete until the blocking findings below are
addressed and re-reviewed.

## Passing findings

- `LoadFile(path)` remains a valid call and defaults to checksum verification.
- `LoadOptions.VerifyChecksum` is a pointer, so `LoadOptions{}` cannot
  silently disable verification.
- `MemoryModeLoad` is the only accepted explicit mode; an unknown mode is
  rejected by `resolveLoadOptions` before `readEmbedFile` opens the path.
- Checksum bypass only bypasses the SHA-256 comparison. `loadBytes` still
  parses header/manifest/metadata and checks vector size, finiteness,
  normalization, and duplicate IDs through `NewMemoryStore`.
- `VerifyFile` calls `loadBytes` without a false flag, therefore retains
  unconditional checksum verification in the implementation.

## Blocking findings

### B1 — README documents a non-existent loader API

`README.md` still uses and lists `embedstore.WithChecksumVerification(true)`.
That symbol does not exist; the planned API is `LoadOptions` with the
`VerifyChecksum` pointer and `MemoryMode`. Copying the documented example
does not compile. The README must show `LoadFile(path)` as the safe default
and document explicit checksum opt-out as diagnostic/trusted-local use only.

**Reproduce:**

```bash
rg -n 'WithChecksumVerification' README.md
go test ./...
```

The first command identifies the stale public example; the repository test
suite cannot compile README snippets, so its success does not validate this
contract.

### B2 — design and requirement documents do not state the exact v1 options contract

`docs/REQUIREMENTS.md` only says `LoadFile` accepts checksum and memory-mode
options. `docs/ARCHITECTURE.md` does not specify the concrete default or the
single supported mode. Neither document states: `nil` checksum option means
verify, `MemoryModeLoad`/empty are valid, unknown modes are rejected, checksum
bypass preserves structural validation, and `VerifyFile` always verifies the
checksum. This is a public API behavior change and the planner explicitly
requires matching documentation.

**Reproduce:**

```bash
rg -n 'LoadOptions|VerifyChecksum|MemoryModeLoad|WithChecksumVerification' README.md docs/ARCHITECTURE.md docs/REQUIREMENTS.md
```

### B3 — acceptance tests do not cover the checksum-disable boundary

`load_options_test.go` covers default rejection and one successful bypass,
but it misses the planned security-boundary cases: zero-valued options reject
a checksum mismatch; explicit `VerifyChecksum: true` rejects it; bypass still
rejects malformed metadata, vector count/shape errors, duplicate IDs,
NaN/Inf/zero vectors, and non-unit vectors. The current truncated-file test
only proves a very early structural rejection.

Add focused table-driven tests using a checksum-recomputed fixture for every
structural corruption so that bypass cannot accidentally grow beyond its
intended scope.

**Reproduce:**

```bash
go test -run 'TestLoadFileChecksum(OptionAndMemoryMode|BypassStillValidatesStructure)' .
```

### B4 — VerifyFile and FileError invariants have no options regression tests

There is no test proving `VerifyFile` rejects a checksum-mismatched file even
though `LoadFile(...VerifyChecksum:false)` accepts it. There is also no test
that file-derived failures from each options path preserve both
`errors.Is(err, ErrInvalidFile)` and `*FileError` path/offset context. These
are explicit acceptance criteria.

**Reproduce:**

```bash
go test -run 'TestLoadFileChecksumOptionAndMemoryMode|TestLoadFileChecksumBypassStillValidatesStructure' .
```

The listed tests complete without exercising `VerifyFile` or inspecting
`FileError`.

## Non-blocking compatibility risk

Changing `LoadFile` from `func(string) (Store, error)` to a variadic function
keeps direct one-argument calls valid, as required by the planner. It does
not preserve callers that store the exported function in an exact function
type, for example:

```go
var load func(string) (embedstore.Store, error) = embedstore.LoadFile
```

This is a Go source compatibility change. It is acceptable only if the
project's compatibility promise is limited to direct calls. Otherwise provide
`LoadFileWithOptions` and retain the old `LoadFile(path string)` declaration.
Document the chosen policy in the handoff/PR.

## Required final verification

After actor changes, run:

```bash
gofmt -w file.go load_options_test.go
go test ./...
go vet ./...
git diff --check
go test -race ./...
```

If the race command still fails due to the known ThreadSanitizer VMA
environment limitation, record the command output and impact in
`tasks/PROGRESS.md`; do not represent it as a passing check.
