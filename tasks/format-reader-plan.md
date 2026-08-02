# File-format reader rotation plan

## Scope

Harden the v1 reader without changing its binary layout or public API.

## Work

1. Add explicit reader limits for file size, manifest bytes, item metadata bytes, item count, and vector dimensions.
2. Check file size with `Stat` before `ReadFile` and reject oversized regular files.
3. Apply every length/count bound before allocations or slicing; use safe arithmetic for vector counts.
4. Add focused regression tests for oversized input and declared manifest/item lengths.
5. Run `gofmt` and the package test suite.
