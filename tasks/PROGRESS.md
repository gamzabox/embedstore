# embedstore 구현 진행 현황

최종 갱신: 2026-08-02 (KST)

## 역할별 계획 문서

| 역할 | 문서 | 상태 |
| --- | --- | --- |
| Root | `tasks/root-plan.md` | 구현 순서와 통합 검증 기준 정의 |
| Planner | `tasks/planner-plan.md` | 요구사항 분해와 수용 기준 검토 완료 |
| Actor | `tasks/actor-plan.md` | 메모리 검색 수직 단위 구현 완료 |
| Evaluator | `tasks/evaluator-plan.md` | 검증 전략 및 미구현 범위 점검 완료 |

## 완료된 구현

- Go module 초기화 (`github.com/embedstore/embedstore`).
- 루트 공개 API: `Manifest`, `Item`, `Store`, `Embedder`, `SearchOptions`,
  `SearchResult`, `Engine`, `DecodeData` 및 공개 sentinel 오류.
- 연속 `float32` 배열을 사용하는 읽기 전용 `MemoryStore`.
- L2 정규화, cosine inner-product 검색, 기본 limit(5), `MinScore`,
  차원 검증, 모델 호환성 검사, 공백 쿼리 거부.
- 결정적 결과 순서와 동시 `SearchVector` 호출 테스트.
- 초기 v1 `.embed` writer/reader/inspect/verify 기반: magic, 버전,
  manifest, item metadata, float32 vectors, SHA-256 checksum 검사.
- 파일 검증 오류에 path와 offset을 제공하는 `FileError`.
- `LoadFile(path)`의 기존 정확한 함수 시그니처를 유지하고, 명시적 loader 옵션은 `LoadFileWithOptions(path, options)`로 분리. checksum 기본 검증과 `VerifyFile`의 무조건 검증을 회귀 테스트로 보장.
- OpenAI Embedder: `openai/<model>` 파싱, API key 옵션·환경변수, dimensions, timeout/HTTP client, 응답 순서·차원 검증, transient retry 및 `httptest` 테스트.
- CLI `inspect`: fast manifest summary plus `--list`/`--id`, text/JSON output, `--pretty`, and conflicting-option/missing-ID tests.

## 검증 결과

| 명령 | 결과 | 비고 |
| --- | --- | --- |
| `gofmt` | 통과 | 현재 Go 소스 포맷 적용 |
| `go test ./...` | 통과 | 메모리 검색/엔진 테스트 포함 |
| `go vet ./...` | 통과 | |
| `git diff --check` | 통과 | |
| `go test -race ./...` | 실행 불가 | ThreadSanitizer가 환경의 unsupported VMA range로 테스트 시작 전 종료 |

## 남은 구현 범위

- 입력 JSON parser/validator: strict 모드, UTF-8·크기·중복 검사,
  canonical JSON 및 결정적 UUID v5 자동 ID.
- OpenAI Embedder: API key 옵션/환경변수, dimensions, timeout, 순차 batch,
  응답 순서 및 retry/backoff 테스트.
- build pipeline: source checksum, vector embedding/정규화, temp 파일 검증,
  atomic rename, overwrite 및 content 전송 경고.
- `--reuse` 병합 build와 content 포함 여부 검증.
- CLI: `validate`, `build`, `search`, `inspect`, `verify` 및 text/JSON 출력
  계약, 0이 아닌 오류 종료.
- file-format golden/corruption/fuzz tests와 CLI/OpenAI HTTP fake tests.
- README 및 요구사항 문서의 실제 CLI/API와의 최종 대조 및 필요한 갱신.

## 설계 확인이 필요한 항목

`README.md`와 상세 build 요구사항은 `--reuse`를 설명하지만,
`docs/REQUIREMENTS.md`의 릴리스 표 및 architecture 확장 계획은 이를
v0.2 범위로 분류한다. 현재 계획은 사용자 요청에 따라 문서에 명시된
기능 전체를 구현 대상으로 유지한다. MVP만을 목표로 변경할 경우 같은
변경에서 README와 요구사항의 범위 표기를 정합화해야 한다.
