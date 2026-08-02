# Actor v2 implementation plan

## Scope

Implement strict-mode validation only in `input.go` and its focused tests.

## Acceptance criteria

1. Normal validation keeps its existing normalization and generated-ID behavior.
2. Strict validation rejects explicit IDs with leading or trailing whitespace.
3. Strict validation rejects explicit IDs that are not in the canonical generated
   UUID v5 form for the item's normalized content and canonical data.
4. Regression tests cover accepted canonical IDs and each rejection path.
5. Run `gofmt`, `go test ./...`, `go vet ./...`, and `git diff --check`.
