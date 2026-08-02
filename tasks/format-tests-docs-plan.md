# File-format rotation B: tests and documentation

## Scope

Strengthen the existing v1 binary-file contract without changing its reader or
writer implementation. The current format is a length-prefixed, sequential
record layout; it does not contain a metadata index.

## Plan

1. Document the actual v1 byte order and sequential item-record layout in the
   architecture document.
2. Add a layout/golden regression test that checks fixed little-endian fields,
   record ordering, vector bytes, and checksum coverage.
3. Add corruption tests for malformed item metadata and trailing bytes with a
   recomputed checksum, so validation is not dependent only on checksum
   mismatch.
4. Add a fuzz target for the full in-memory loader and seed it with valid and
   malformed files; the acceptance condition is no panic.
5. Run focused format tests, then the full Go test suite.

## Non-goals

Adding a metadata index or a new format version is intentionally out of scope.
Those changes require a separately versioned file-format design and reader /
writer implementation work.
