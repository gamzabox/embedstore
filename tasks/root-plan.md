# Root agent implementation plan

## Objective

Deliver the v0.1 MVP defined in `README.md`, `docs/ARCHITECTURE.md`, and
`docs/REQUIREMENTS.md`, with the CLI and public Go API sharing the same
reader and in-memory search engine.

## Work sequence

1. Consolidate the planner's component plan and evaluator's acceptance matrix.
2. Establish the Go module, public data types/errors, normalization, input
   validation, and deterministic IDs.
3. Implement a versioned, checksummed binary file writer/reader and its
   corruption checks.
4. Implement the immutable memory store and engine, then test deterministic
   Top-K search and model/dimension compatibility.
5. Implement the OpenAI HTTP embedder and build pipeline with sequential
   batching, retries, validation, and atomic output replacement.
6. Wire `validate`, `build`, `search`, `inspect`, and `verify` commands to
   those shared components and update user-facing documentation where an
   implementation choice needs to be explicit.
7. Run formatting, unit tests, vet, race tests, and diff checks; address every
   failure before handoff.

## Acceptance checks

- Unit and corruption tests cover all public sentinel errors and file
  validation paths.
- No API key or authorization value can enter file contents or error text.
- Existing output survives any failed build; replacement requires
  `--overwrite`.
- `go test ./...`, `go vet ./...`, `go test -race ./...`, and
  `git diff --check` pass.
