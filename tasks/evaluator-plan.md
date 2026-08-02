# Evaluator verification plan

## Baseline assessment (2026-08-02)

The repository currently contains the specification documents, README, `AGENTS.md`,
and task plans only. It has no `go.mod`, Go packages, command implementation, or
tests. Consequently, `go test ./...`, `go vet ./...`, and `go test -race ./...`
cannot yet be executed; this is an implementation gap, not a passing baseline.

The evaluator must review changes against `README.md`, `docs/ARCHITECTURE.md`, and
`docs/REQUIREMENTS.md` as one public contract. Any deliberate contract change must
update the document owning that contract in the same change.

## Acceptance gates and ownership boundaries

1. **Input and public API gate**
   - Test dataset object requirements; empty/malformed JSON; UTF-8; record and
     content-size bounds; absent/non-serializable `data`; duplicate explicit and
     generated IDs; and deterministic UUID v5 generation from normalized content
     and canonical JSON data.
   - Test all exported sentinel errors through `errors.Is`, `FileError` path and
     byte offset, `DecodeData[T]`, and API examples as compile tests.
   - Confirm the library emits no logs unless given an optional caller logger.

2. **Embedding/build gate (all tests use `httptest` or a fake Embedder)**
   - Assert provider/model parser accepts only normalized `<provider>/<model>` and
     rejects unsupported providers before requests are made.
   - Assert OpenAI request body, optional dimensions propagation, response ordering,
     dimensions validation, 30s/5 default settings, configured HTTP client,
     timeout/context cancellation, and that keys/Authorization/content never occur
     in errors or logs.
   - Exercise sequential batching at both 100 item and 100,000-token boundaries,
     including a one-item boundary and oversized individual input policy.
   - Exercise retry only for 429, 500, 502, 503, 504 and temporary transport errors;
     assert exponential attempts and no retry for permanent 4xx.
   - Assert normalization, zero/NaN/Inf rejection, actual response dimension in the
     manifest, output-exists refusal, overwrite-only atomic replacement, no final
     partial file after every injected failure, warning before external transmission,
     and no secret material in output bytes.
   - Test `--reuse`: dataset-name mismatch, absent source content, input precedence,
     retention of old-only IDs, new dataset version, full re-embedding of every
     merged item, and allowed old/new embedding or dimension differences.

3. **File format and loader gate**
   - Require writer/reader golden fixtures and document exact fixed endianness,
     magic bytes, v1 header/index layout, checksum coverage, and deterministic
     metadata key ordering.
   - Round-trip manifests (all required fields), `contentIncluded` true/false,
     raw metadata preservation, contiguous float32 layout, source checksum, and
     stable search/input order.
   - Mutate golden bytes to test truncation, bad magic/version/header length,
     impossible/overflowing count or offset/length, out-of-range metadata,
     malformed metadata JSON, count/dimension mismatch, checksum mismatch,
     NaN/Inf/zero vectors, and oversized declarations. Each must fail safely with
     the proper sentinel error and a useful `FileError` location when available.
   - Verify `inspect` performs only shallow manifest/index access while `verify`
     rechecks every required range, metadata block, vector, and checksum.
   - Add Go fuzz targets for the JSON parser and file reader; seed them with valid
     golden files and the corruption cases above.

4. **Search gate**
   - Check L2 normalization of input query, dot-product cosine scores, dimension
     mismatch, empty/zero/NaN/Inf query rejection, default limit 5 for zero value,
     `MinScore`, limits larger than count, and deterministic ties (stored order then
     ID).
   - Assert the Top-K implementation retains only K candidates (not a full sort)
     through focused heap unit tests and code review.
   - Check `NewEngine` rejects model and dimension incompatibility with the required
     sentinels; search embeds exactly one query and handles cancellation.
   - Run concurrent `SearchVector` tests under the race detector on a loaded,
     read-only store.

5. **CLI contract gate**
   - Add black-box tests for `validate`, `build`, `search`, `inspect`, and `verify`:
     required flags, defaults, invalid flag values, non-zero failures, actionable
     stderr, and no sensitive content/credentials in errors/logs.
   - Compare text and JSON output fields/order against the documented examples.
     Cover JSON pretty mode, data always present, search content/vector inclusion,
     vector-with-text rejection, omitted stored content error, `--min-score`, and
     manifest-derived search provider selection/unsupported provider error.
   - Cover inspect summary/list/id modes, list+id conflict, content absence, and
     verify text/JSON success and corrupted-file failures. Confirm inspect does
     not trigger full checksum/vector validation.
   - Verify `--log-format text|json` and the documented error/warn/info/debug levels
     without leaking API credentials, authorization headers, or sensitive content.

## Required commands before handoff

Run after adding a Go module and tests, from the repository root:

```bash
gofmt -w $(git diff --name-only -- '*.go')
go test ./...
go vet ./...
go test -race ./...
go test -run '^$' ./...
go test -fuzz=Fuzz -fuzztime=30s ./internal/fileformat ./internal/input
git diff --check
git status --short
```

For performance evidence, run benchmarks without treating their absolute timing as
a correctness gate:

```bash
go test -run '^$' -bench 'Benchmark(Search|Load)' -benchmem ./...
```

The final evaluator report must state each command's exit status, any skipped
command and reason, added fixtures/fuzz seeds, and remaining release checks
(`staticcheck`, cross-platform GoReleaser artifacts, and `checksums.txt`).

## Risks and decisions requiring planner confirmation

- The functional requirements describe `--reuse` as part of `build`, while the
  release table places merge build in v0.2.0. Resolve this discrepancy explicitly
  before implementation; the conservative evaluator interpretation is to implement
  `--reuse` because it is specified in the detailed build contract and README.
- Token accounting is called “estimated” but its exact algorithm and behavior for a
  single item above `--max-batch-tokens` are unspecified. Define and document both
  before test fixtures are finalized.
- “UTF-8” and “serializable data” need concrete Go/JSON acceptance semantics;
  standard JSON decoding has no invalid UTF-8 string representation after decode.
  Define whether raw input bytes are validated before decoding and whether data must
  exclude only unsupported JSON forms (for example non-finite numbers).
- Architecture says `LoadFile` validates on load while the API offers optional
  checksum verification and `inspect` must be shallow. Specify the reader modes
  and which structural/vector checks each mode performs so loader and inspect tests
  do not encode conflicting expectations.
- Fixed namespace UUID value, normalization rule for content, file-size/count
  ceilings, file binary layout, and source checksum input are contracts that need
  concrete constants before golden fixtures can be authoritative.
