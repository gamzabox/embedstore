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
- 입력 JSON parser/validator: strict 모드, UTF-8·크기·중복 검사, canonical JSON 및 결정적 UUID v5 자동 ID, validation summary.
- build pipeline: 순차 배치·토큰 상한, vector 정규화, temp 파일 검증·원자적 rename, overwrite, content 전송 경고 및 `--reuse` 병합 재임베딩.
- CLI `validate`, `build`, `search`, `inspect`, `verify`: text/JSON 출력과 오류 종료 계약의 end-to-end 테스트.
- 파일 포맷 binary-layout/golden 성격의 계약 테스트, 손상 파일 테스트, loader·input parser fuzz 테스트, CLI/OpenAI HTTP fake 테스트.

## 검증 결과

| 명령 | 결과 | 비고 |
| --- | --- | --- |
| `gofmt` | 통과 | 현재 Go 소스 포맷 적용 |
| `go test ./...` | 통과 | 메모리 검색/엔진 테스트 포함 |
| `go vet ./...` | 통과 | |
| `git diff --check` | 통과 | |
| `go test -race ./...` | 실행 불가 | ThreadSanitizer가 환경의 unsupported VMA range로 테스트 시작 전 종료 |
| `go test -run=^$ -fuzz=FuzzParseDataset -fuzztime=10s .` | 통과 | 5개 seed 기준 약 8.2만 입력, 새 흥미 입력 75개 |

## 최근 완료

- GitHub Actions CI와 GoReleaser 설정: 모든 push/PR의 format·test·vet·diff·staticcheck·Linux race 검사, `v*` 태그의 GitHub Release 및 Linux amd64/arm64·macOS amd64/arm64·Windows amd64 archive와 SHA-256 `checksums.txt` 생성.
- planner → actor → evaluator 검토 완료: 차단 이슈 없음. release checkout은 credential을 남기지 않고 GoReleaser에만 명시적 `GITHUB_TOKEN`을 전달한다.
- 로컬 검증: CI format 명령, `go test ./...`, `go vet ./...`, `git diff --check`. 이 환경에는 YAML validator와 GoReleaser CLI가 없어 workflow YAML 및 `goreleaser check`/snapshot release는 GitHub Actions의 최초 실행에서 확인해야 한다.

## 남은 구현 범위

- 1천/1만/5만 항목과 512/1,024/1,536 차원의 검색 benchmark.
- v0.2 후속 기능: interactive shell과 evaluate 명령.
- v0.3 이후: labels 필터, batch query, mmap, 추가 provider, ANN/HNSW 등.

## 설계 확인이 필요한 항목

`--reuse`는 v0.1 구현 범위에 포함되도록 README, Architecture, Requirements의 범위를 정합화했다. 남은 후속 기능은 구현을 시작할 때 각 릴리스 표와 함께 상태를 갱신한다.
