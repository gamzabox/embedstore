# LoadFile options planner plan

## Scope

Add a backward-compatible public loader-options API so callers can choose
checksum verification and request the MVP memory mode explicitly. The existing
`LoadFile(path string) (Store, error)` call must remain source-compatible and
retain its current safe behavior: complete verification, including checksum,
followed by an immutable in-memory store.

The work covers the root package loader and its public documentation. It does
not change the v1 `.embed` byte format, add mmap, or change CLI command
semantics.

## Proposed public API

```go
type MemoryMode string

const (
    MemoryModeLoad MemoryMode = "load"
)

type LoadOptions struct {
    // VerifyChecksum controls SHA-256 verification. Its zero value preserves
    // the safe default and verifies the checksum.
    VerifyChecksum *bool

    // MemoryMode selects file loading behavior. Empty means MemoryModeLoad.
    // MVP supports only MemoryModeLoad.
    MemoryMode MemoryMode
}

func LoadFile(path string, options ...LoadOptions) (Store, error)
```

`LoadFile(path)` remains valid through a variadic options parameter. At most
one `LoadOptions` value is accepted; more values return `ErrInvalidQuery` (or
an explicitly documented public option-validation error if the actor finds an
existing suitable sentinel). `MemoryModeLoad` is the only accepted non-empty
mode in v1; unknown modes are rejected before opening/parsing the file.

`VerifyChecksum` is a pointer deliberately: `nil` means the secure default
`true`, while `Bool(false)` allows an explicit opt-out. This avoids making a
plain `bool` zero value silently disable integrity verification. Skipping the
checksum must still validate every structural bound, metadata record, vector,
normalization invariant, and duplicate ID.

`VerifyFile` remains an unconditional full verifier and must not gain an
option that can skip checksum verification. `InspectFile` remains its current
fast header/manifest reader. CLI `search`, `verify`, and reuse build continue
to use the safe default loader behavior.

## Acceptance criteria

1. Existing `LoadFile(path)` tests and callers compile unchanged and still
   reject a checksum mismatch with `errors.Is(err, ErrInvalidFile)`.
2. `LoadFile(path, LoadOptions{})` performs checksum verification, proving the
   zero option value is safe.
3. `LoadFile(path, LoadOptions{VerifyChecksum: boolPtr(false)})` loads a file
   whose only defect is a checksum mismatch, but rejects the same file when
   the byte layout, metadata, vector shape, duplicate IDs, NaN/Inf/zero, or
   unit-length invariant is invalid.
4. `LoadFile(path, LoadOptions{VerifyChecksum: boolPtr(true)})` rejects a
   checksum mismatch.
5. Empty `MemoryMode` and `MemoryModeLoad` both succeed; an unknown memory
   mode is rejected deterministically without exposing content or a partial
   `Store`.
6. Supplying multiple option values is rejected deterministically (if the
   variadic proposal is used).
7. Error wrapping preserves `FileError` path/offset for file-derived
   validation failures and `errors.Is` sentinel behavior.
8. Add unit tests before implementation and run `gofmt`, `go test ./...`,
   `go vet ./...`, `git diff --check`, and `go test -race ./...` because this
   changes a public file reader. If race cannot run due to the existing VMA
   environment limitation, record it in `tasks/PROGRESS.md`.

## Documentation changes

- `README.md`: add a concise Go API example showing `LoadFile(path)` and
  explicit checksum opt-out only for trusted/local diagnostic use; state that
  the default verifies checksum and loads into memory.
- `docs/ARCHITECTURE.md`: describe the `LoadOptions` default, `MemoryModeLoad`,
  and that mmap is deferred to a later format-compatible version.
- `docs/REQUIREMENTS.md`: replace the generic LoadFile option statement with
  the exact supported v1 contract, including `VerifyChecksum` default and
  `MemoryModeLoad`.
- `tasks/PROGRESS.md`: record the completed API contract, tests, and any race
  environment limitation.

## Risks and decisions

- A bare `bool` would make `LoadOptions{}` unsafe, so use `*bool` (or an
  equivalent explicit tri-state) rather than `bool`.
- Checksum bypass weakens integrity guarantees; it must never become a CLI
  default and should not alter `VerifyFile`.
- Variadic `LoadOptions` preserves the documented one-argument API. An
  alternative `LoadFileWithOptions` would avoid multiple-option validation but
  leaves the required LoadFile options API less direct; prefer the variadic
  API unless evaluator identifies a compatibility issue.
- Memory mapping, partial metadata loads, and a new file format/index are
  explicitly excluded; only the existing complete in-memory load mode is
  implemented.

## Handoff

Actor should add the failing tests first, implement only this API surface and
documentation, and record test output in the progress log. Evaluator should
focus on zero-value safety, bypass scope, sentinel/FileError behavior, and
whether CLI paths accidentally pass checksum-disabled options.
