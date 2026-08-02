# LoadFile API compatibility plan

## Scope

Replace the current variadic public loader declaration with an exact legacy
signature, and move option handling to a separately named API:

```go
func LoadFile(path string) (Store, error)
func LoadFileWithOptions(path string, options LoadOptions) (Store, error)
```

This follows the evaluator's compatibility finding and the user's explicit
decision. It preserves both direct calls and uses where callers assign
`LoadFile` to `func(string) (Store, error)`. The v1 file format, complete
in-memory loading behavior, CLI semantics, and existing option meanings are
not changed.

## Exclusions

- No mmap or additional memory modes.
- No change to `LoadOptions`, checksum bypass semantics, file layout, or
  loader validation limits.
- No CLI option for disabling checksum verification.

## Required implementation behavior

1. `LoadFile(path)` delegates to `LoadFileWithOptions(path, LoadOptions{})`.
2. `LoadFileWithOptions` accepts one value (not a variadic list), validates
   `MemoryMode`, reads the file, and applies `VerifyChecksum` exactly as the
   current loader does.
3. A zero `LoadOptions` value and `VerifyChecksum: nil` verify the checksum;
   an explicit false value bypasses only the checksum comparison, never
   structural, metadata, vector, normalization, or duplicate-ID validation.
4. `VerifyFile` remains an unconditional checksum verifier.
5. Invalid `MemoryMode` returns an `ErrInvalidQuery`-wrapping error before
   parsing the file. File-derived errors retain `FileError` and `errors.Is`
   behavior.

## TDD and acceptance tests

Actor should first adjust/add tests for:

- exact function type assignment:
  `var load func(string) (Store, error) = LoadFile`;
- default `LoadFile(path)` rejection of checksum mismatch;
- `LoadFileWithOptions` zero/default, explicit true, and explicit false
  checksum cases;
- supported/unsupported memory mode behavior;
- checksum-bypassed structural corruption still returning `ErrInvalidFile`;
- `VerifyFile` rejecting a checksum-mismatched fixture even when the options
  API can load that fixture.

Run `gofmt`, `go test ./...`, `go vet ./...`, `git diff --check`, and
`go test -race ./...` (record the known environment limitation if it recurs).

## Documentation changes

- README Go API examples must present `LoadFile(path)` as the normal safe
  loader and use `LoadFileWithOptions` for the explicitly trusted diagnostic
  checksum opt-out.
- ARCHITECTURE must list both APIs and explain their safe default.
- REQUIREMENTS must name `LoadFileWithOptions` as the options API while
  retaining `LoadFile` as the stable no-options entry point.
- PROGRESS must record the compatibility correction and validation results.

## Risks

- Renaming only the implementation but leaving documentation with a variadic
  `LoadFile` signature would recreate the evaluator's public-contract block.
- `LoadFileWithOptions` must not accidentally accept multiple logical option
  values; a single `LoadOptions` value intentionally removes that ambiguous
  API surface.

## Handoff

Actor implements only this API split plus tests and documentation. Evaluator
checks exact function-type compatibility, checksum-default safety, sentinel
and `FileError` behavior, and that no CLI path adopts checksum bypass.
