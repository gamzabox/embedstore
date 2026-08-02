# Actor implementation plan

## Scope

Implement and validate the root-package, in-memory search vertical slice while the parallel workstream owns input parsing, file-format/loader, CLI, and OpenAI transport.

## Plan

1. Define and test public manifests, errors, store/embedder interfaces, and metadata decoding.
2. Implement immutable contiguous-vector in-memory cosine search with default limit, filtering, deterministic ties, and concurrent safety.
3. Implement compatible embedder checks and query search in `Engine`.
4. Integrate against the shared loader and run formatting, test, vet, race, and diff checks.

## Integration contract

The file loader constructs `MemoryStore`; CLI and Go applications use the same `Store.SearchVector` and `Engine.Search` behavior.
