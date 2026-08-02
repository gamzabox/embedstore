# Evaluator v2 CLI end-to-end test plan

## Permission smoke test

This document is intentionally created with `apply_patch` by the refreshed evaluator agent. Its presence verifies that the agent can write to the repository workspace.

## Scope

Review the command-line contracts in `docs/REQUIREMENTS.md` without changing Go source. The next implementation work should add process-level tests which execute the compiled CLI against temporary JSON inputs, `.embed` fixtures, and an `httptest` OpenAI endpoint.

## Exact test matrix

### validate

1. A valid dataset exits 0 and emits the required text summary: item count, duplicate IDs, empty content, invalid records, and estimated tokens.
2. `--format json` and `--output json` emit one parseable JSON summary; unsupported format and output values exit non-zero.
3. Invalid JSON, missing top-level fields, duplicate explicit or generated IDs, and an oversized content value exit non-zero with an actionable error.
4. `--strict` has dedicated acceptance tests once its additional validation policy is specified; the test must prove it rejects an input accepted by non-strict mode.

### build

1. Run build against an `httptest` OpenAI embeddings server and verify output is loadable, manifest embedding/dimensions are correct, vectors are normalized, and request batches obey `--batch-size` and `--max-batch-tokens`.
2. Verify default API-key environment lookup, explicit `--api-key` precedence (if exposed by CLI), dimensions request forwarding, timeout/cancellation, and retry count for 429/5xx responses.
3. Confirm output collision fails without `--overwrite`; overwrite replaces only after complete validation, while an embedding failure leaves the old output byte-for-byte intact.
4. Confirm `--reuse` preserves reuse-only IDs, replaces colliding IDs with input records, re-embeds every merged item, rejects dataset mismatch and content-omitted reuse files.
5. Assert stderr contains the content-transmission warning and never leaks the API key.

### search

1. Build a fixture with an `httptest` provider, invoke search, and assert the manifest embedding selects the request model/dimensions.
2. Cover default limit, explicit limit, `--min-score`, deterministic ranking, text output fields, JSON schema, `--pretty`, `--include-content`, and JSON-only `--include-vector`.
3. A content-omitted file with `--include-content`, unsupported manifest provider, missing key, embedding dimension mismatch, and invalid output options must exit non-zero.

### inspect and verify

1. Invoke inspect in summary, `--list`, and `--id` modes for text/JSON/pretty output; assert `--list` plus `--id` fails.
2. Assert inspect does not require a full checksum scan (use a fixture with a damaged trailing checksum or a test seam that detects `VerifyFile` usage).
3. Invoke verify on a valid file and fixtures with bad magic/version, truncation, checksum mismatch, malformed metadata, invalid vector, and invalid offsets; each invalid fixture exits non-zero and includes path/offset where available.

## Test harness design

- Prefer a `TestMain`/helper-process pattern or `go build -o <temp>/embedstore ./cmd/embedstore`, then use `exec.CommandContext`; do not call unexported command helpers.
- Give each test a temporary directory and use `httptest.Server`; set only test-specific environment variables.
- Capture stdout and stderr separately, assert exit status, and parse JSON rather than comparing formatting-sensitive raw JSON.
- Keep network tests local: no real OpenAI calls and no real credentials.

## Completion criteria

All tests run through `go test ./...`; update `tasks/PROGRESS.md` with command coverage and any intentionally deferred contract after the implementation lands.
