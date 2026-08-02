# Rotation: file-format hardening

## Scope

Harden v1 reader/writer boundary checks without changing the public on-disk
layout. Add regression coverage for malformed manifest and metadata lengths,
semantic manifest validation, and defensive copies from the read-only store.

## Steps

1. Add failing cases for malformed lengths and invalid manifest fields.
2. Apply explicit, overflow-safe bounds checks before parsing or allocation.
3. Ensure loaded metadata cannot be mutated through caller-visible results.
4. Run gofmt and focused package tests.
