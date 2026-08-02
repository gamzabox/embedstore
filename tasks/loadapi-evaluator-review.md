# LoadFile API compatibility evaluator review

## Verdict

No blocker found. The implemented API split matches the approved plan and
preserves the exact legacy function type of `LoadFile`.

## Checked behavior

- **Public compatibility — pass.** `LoadFile` is declared as
  `func(string) (Store, error)`, delegates to `LoadFileWithOptions`, and the
  regression test assigns it to that exact function type.
- **Options API — pass.** `LoadFileWithOptions` accepts exactly one
  `LoadOptions` value. A zero value and an explicit `VerifyChecksum: true`
  verify the checksum; explicit false skips only the checksum comparison.
  Unsupported memory modes fail before file I/O with an `ErrInvalidQuery`
  wrapping error.
- **File safety — pass.** The checksum-bypass test still rejects truncated
  input. `loadBytes` continues to validate header/version, lengths, manifest,
  metadata, vector bounds/normalization, and duplicate IDs through
  `NewMemoryStore`.
- **Checksum and errors — pass.** A corrupted checksum produces both
  `errors.Is(err, ErrInvalidFile)` and `errors.As(err, *FileError)` for the
  default loader, explicit default/true options, and `VerifyFile`.
  `VerifyFile` calls the loader without the bypass flag, so its full checksum
  validation remains unconditional.
- **Call paths — pass.** CLI, build reuse, and existing callers use the safe
  no-options `LoadFile(path)` path; no CLI checksum-bypass option was added.
- **Documentation — pass.** README, architecture, requirements, and progress
  consistently name `LoadFileWithOptions` as the explicit-options API and
  state the default checksum behavior.

## Validation run

```text
go test ./...  # pass
go vet ./...   # pass
git diff --check # pass
```

`go test -race ./...` was attempted but cannot run in this environment:

```text
FATAL: ThreadSanitizer: unsupported VMA range
FATAL: Found 39 - Supported 48
```

This is an environment/runtime limitation rather than a test failure. The non-race suite passes; run the race suite on a supported VMA layout before a release that requires race-detector assurance.

## Non-blocking observations

- `MemoryModeLoad` currently selects the same in-memory implementation as an
  empty memory mode, which is consistent with the MVP scope.
- `loadBytes` has a variadic internal checksum argument for existing test
  convenience. It is unexported and does not affect the restored public API
  compatibility.
