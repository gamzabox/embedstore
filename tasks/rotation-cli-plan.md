# Rotation 1: CLI validation

## Scope

- Implement concrete `validate --strict` input-normalization checks in `input.go`.
- Add unit coverage for strict-only rejection cases.
- Add process-level CLI coverage for strict validation and actionable validate/search error paths.

## Strict contract

Strict mode rejects values that the normal parser would silently normalize:

1. `datasetName` and `datasetVersion` with leading or trailing whitespace.
2. Explicit IDs with leading/trailing whitespace or control characters.
3. Content with leading/trailing or repeated whitespace.

Normal validation keeps its existing normalization and backward-compatible behavior.

## Validation

- Run `gofmt` on changed Go files.
- Run relevant package tests, then `go test ./...` if the shared worktree is stable.
