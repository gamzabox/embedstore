# embedstore 전체 구현 계획

## 1. 현황과 계획 전제

2026-08-02 기준 저장소에는 `README.md`, `docs/ARCHITECTURE.md`,
`docs/REQUIREMENTS.md`, 그리고 빈 초안인 `tasks/actor-plan.md`만 있다. `go.mod`,
Go 소스, 테스트, CLI, CI 설정은 없으므로 아래 계획은 기존 기능 보완이 아니라
새 Go 모듈을 처음부터 구현하는 순서다.

문서상 구현 기준은 README/Architecture/Requirements이며, 공개 파일 포맷은
format version 1로 고정한다. 호환되지 않는 포맷 변경은 하지 않는다.

### 범위 결정 필요: `--reuse`

README와 build 요구사항은 `--reuse` 병합 build를 명시하지만, Architecture의
"MVP 후" 문장과 Requirements의 릴리스 표는 이를 v0.2.0으로 배치한다. 본
작업이 "문서의 전체 기능" 구현을 뜻하므로 기본 계획에는 `--reuse`를 포함한다.
릴리스 MVP만을 엄격히 구현하기로 변경될 경우에는 단계 7의 reuse 부분만
v0.2.0 후속 작업으로 분리해야 하며, README/Requirements/Architecture의 범위
기술을 같은 변경에서 일치시킨다.

## 2. 구현 순서와 담당 분리

동시에 수정 충돌이 나지 않도록 foundation → format/search → provider/build/CLI
순서로 진행한다. actor는 각 단계에서 **실패 테스트를 먼저** 추가하고 최소
구현으로 통과시킨다. evaluator는 각 완료 단위를 독립적으로 검토하고 회귀
테스트를 추가한다.

| 단계 | 주 담당 | 산출물 | 선행 조건 |
| --- | --- | --- | --- |
| 0 | planner | 이 계획, API/포맷 결정 기록 | 없음 |
| 1 | actor | Go 모듈, 공개 도메인 타입/오류, 입력 파서·검증 | 0 |
| 2 | actor | 정규화, Top-K, `MemoryStore`, `Engine` | 1 |
| 3 | actor | v1 writer/reader/verifier, `LoadFile` | 1, 2 |
| 4 | actor | OpenAI `Embedder` HTTP client | 1 |
| 5 | actor | build orchestration(배치·원자 쓰기·reuse) | 1, 3, 4 |
| 6 | actor | Cobra/flag 기반 CLI: validate/build/search/inspect/verify | 3–5 |
| 7 | evaluator | 계약·손상 파일·race·API/CLI 회귀 검증과 문서 대조 | 각 단계 후, 최종 |
| 8 | actor + evaluator | fuzz/benchmark/CI·릴리스 기반, 전체 품질 게이트 | 6, 7 |

권장 디렉터리 경계는 Architecture를 따른다.

```text
cmd/embedstore/          main
internal/cli/            command parsing, renderers, exit handling
internal/input/          input JSON parsing, canonical JSON, UUID v5 validation
internal/fileformat/     v1 binary writer/reader/quick inspect/full verify
internal/heap/           bounded Top-K heap
internal/normalize/      vector normalization/dot product
embedding/               Embedder-facing shared helpers (필요 시)
embedding/openai/        HTTP OpenAI Embeddings client
root package (*.go)      public Store, Engine, Manifest, Item, errors, LoadFile
```

`internal/fileformat`은 공개 타입을 순환 import하지 않도록 wire 타입을 두거나,
루트 공개 타입만 별도의 의존성 없는 `model` 패키지로 분리하는 설계를 단계 1에서
확정한다. 외부에 노출되는 API는 계속 루트 `embedstore` 패키지에 둔다.

## 3. 단계별 구현 및 수용 기준

### 단계 1 — 모듈, 모델, 입력 검증, 공개 오류

1. `go.mod`를 Architecture의 모듈 경로로 초기화하고 패키지/명령 진입점을 만든다.
   UUID v5 구현에 필요한 의존성은 최소화하고 `go.sum`을 고정한다.
2. 공개 `Manifest`, `Item`, `SearchOptions`, `SearchResult`, `Store`, `Embedder`,
   `DecodeData[T]`, sentinel 오류와 path/offset을 보유한 `FileError`를 정의한다.
3. 입력 래퍼(`datasetName`, `datasetVersion`, `items`)를 파싱한다. 임의 JSON
   metadata는 `json.RawMessage`로 보존하되, 자동 ID 계산에는 재현 가능한 key
   정렬 canonical JSON을 사용한다. content 정규화 규칙(최소한 Unicode/공백
   처리 범위)을 코드와 테스트에서 명확히 고정한다.
4. non-empty 필드, UTF-8, serializable JSON, record-count/content-byte 상한,
   명시/자동 ID의 중복, `id` 길이(특히 strict 모드)의 오류를 구현한다. 검증
   통계(항목·중복·빈 content·잘못된 레코드·추정 토큰)를 만들고 CLI renderer가
   사용하게 한다.

수용 테스트:

- 동일 content + 순서가 다른 `data` key는 같은 UUID v5; content/data 변경은 다른
  자동 ID; 명시 ID는 그대로 유지한다.
- 빈/잘린 JSON, 누락 top-level 필드, 빈 items, 빈 content/data, 중복 ID, invalid
  UTF-8, 상한 초과가 각각 실패한다.
- `errors.Is`가 모든 공개 sentinel에 대해 동작하고 `DecodeData` 성공/실패를
  확인한다.

### 단계 2 — 수학과 메모리 검색

1. float32 L2 normalization은 입력을 불필요하게 변경하지 않는 정책을 문서화하고,
   zero/NaN/Inf는 오류로 거부한다. dot product와 차원 확인을 추가한다.
2. 연속 `[]float32`를 소유하는 read-only `MemoryStore`를 만들고 생성 시 벡터 수,
   dimensions, 정상 정규화 여부를 검증한다.
3. bounded min-heap으로 `O(N log K)` `SearchVector`를 구현한다. score 내림차순,
   동점은 원래 index, 그 다음 ID의 결정적 순서이며 `Limit == 0`은 5다.
4. `Engine`은 한 텍스트를 embedder에 전달하고 정확히 한 쿼리 벡터를 받아
   normalize한 뒤 store 검색한다. `NewEngine`은 embedding ID/dimensions 불일치를
   `ErrModelMismatch`/`ErrDimensionMismatch`로 거부한다.

수용 테스트:

- 3-4-5 벡터 정상화, zero/NaN/Inf 거부, cosine/dot의 기대값, 쿼리 차원 불일치,
  빈 query/빈 embedder 응답을 검증한다.
- top-k, min score, limit 0, limit>count, 동점 tie-break, rank 재번호를 검증한다.
- `go test -race ./...`에서 다수 goroutine의 동시 `SearchVector`가 race 없이
  같은 결과를 반환한다.

### 단계 3 — `.embed` v1 파일 포맷

1. writer/reader가 공통으로 쓰는 정확한 binary layout을 `ARCHITECTURE.md`에
   보완한다: magic 값과 길이, format version과 header-length의 타입·little endian,
   manifest/index/metadata/vector/checksum의 정확한 경계, index entry 구조,
   SHA-256 표현 형식, max file/item/dimension/metadata size 상한.
2. writer는 manifest의 `formatVersion=1`, `vectorType=float32`, `normalized=true`,
   실제 dimensions/item count/contentIncluded/source checksum과 deterministic
   metadata JSON을 기록하고 magic부터 checksum 직전까지 SHA-256을 쓴다.
3. full reader/verify는 **할당 전** file size, header, count, all offset/length,
   multiplication overflow를 확인한 후 manifest/metadata/vector/checksum을
   검증한다. vector는 finite이며 non-zero이어야 한다. 오류는 `FileError`로 path 및
   가능한 offset을 포함하고 `ErrInvalidFile` 또는 `ErrUnsupportedVersion`을 감싼다.
4. quick inspect는 header/manifest/index의 필요한 범위만 읽고 checksum 재계산과
   전 vector scan을 절대 호출하지 않는다. `LoadFile`은 full verification 뒤
   `MemoryStore`로 적재한다. 선택 가능한 load option의 memory mode는 MVP에서는
   일반 메모리만 받아야 한다.

수용 테스트:

- writer → LoadFile → search round-trip과 고정 `createdAt` injection을 이용한
  golden file byte 호환성을 확인한다.
- magic/version/header/length/offset/count overflow, truncate, checksum mismatch,
  bad manifest/metadata, metadata count mismatch, wrong vector bytes, NaN/Inf/zero
  vector마다 실패 sentinel 및 FileError 세부를 검증한다.
- inspect instrumentation 또는 test reader로 full checksum/vector scan을 하지
  않음을 확인한다. fuzz target은 reader를 panic 없이 종료시킨다.

### 단계 4 — OpenAI provider

1. `embedding/openai`에 옵션형 client(API key, model, dimensions, timeout,
   max-retries, HTTP client, optional safe logger)를 구현한다. API key는 option 또는
   `OPENAI_API_KEY`에서 얻되 오류/로그/manifest에 넣지 않는다.
2. `<provider>/<model>`을 엄격 파싱해 openai 외 provider를 구분한다. dimensions는
   API request에만 전달하고, 지원 모델/범위를 검증하며 response vector의 실제
   길이와 일관성을 확인한다.
3. input 배열과 OpenAI response `index`를 사용해 response 순서를 원래 input
   순서로 복원하고, 누락/중복/out-of-range index를 오류 처리한다.
4. context 및 request timeout, 429/500/502/503/504와 임시 network error의 capped
   exponential backoff를 구현한다. 영구 4xx(429 제외)와 malformed response는 즉시
   실패한다. test-friendly clock/sleeper 또는 retry delay injection을 둔다.

수용 테스트(`httptest`; 실제 API 호출 금지):

- Authorization 요청은 보내되 테스트의 로그/오류에는 key가 없고, body의 model,
  input, optional dimensions가 맞다.
- out-of-order 성공 response가 input 순서로 복원되고, 차원/response-count/index
  오류가 실패한다.
- retry 대상별 횟수와 non-retry 4xx, canceled context, timeout, temporary network
  error를 검증한다.

### 단계 5 — build 서비스

1. CLI와 테스트가 주입 가능한 `Embedder`를 공유하는 build service를 만든다.
   validation → optional merge → conservative sequential batch planning → Embed →
   dimension consistency → normalization → writer → full verify → atomic rename의
   순서를 보장한다.
2. 배치는 `batch-size=100`, `max-batch-tokens=100000` 중 먼저 도달한 지점에서
   나눈다. 토큰 추정 정책을 하나로 고정하며, 단일 항목이 상한을 초과할 때의
   명확한 오류를 정한다.
3. output 존재 시 `--overwrite` 없이는 API 호출 전 실패한다. 같은 output dir에
   안전한 임시 파일을 만들며 writer/verify/rename 어느 단계가 실패해도 기존
   output은 보존하고 temp를 정리한다. content가 external API에 전송됨을 warn
   logger로 알리되 content 자체는 log에 쓰지 않는다.
4. `--reuse`는 verified load 후 `contentIncluded` 및 dataset name을 검사하고,
   input ID 우선/기존 단독 ID 유지의 안정적 순서로 metadata+content를 병합한다.
   기존 vector는 사용하지 않고 병합한 전체 항목을 새 embedder에 보낸다.

수용 테스트(fake Embedder):

- batch count/token splitting은 순차 호출·입력 순서를 보장한다; dimensions의
  request/actual-response 불일치, zero vector, partial provider 실패는 output을
  만들지 않는다.
- atomicity: 기존 output 보존, no-overwrite 실패, overwrite 성공, writer/verify
  실패 후 temp 정리 및 final 미변경을 검증한다.
- include-content true/false manifest 및 sensitive field 미기록, reuse의 dataset
  mismatch/content missing/error, input-wins/old-only preservation/전체 reembed를
  검증한다.

### 단계 6 — CLI와 출력 계약

1. root command와 `validate`, `build`, `search`, `inspect`, `verify`를 작성한다.
   각 필수 flag/기본값/enum/range/mutual-exclusion을 Requirements 표대로 검증한다.
   모든 user-facing 오류는 stderr, non-zero exit이며 actionable context만 포함한다.
2. `validate`: `--strict`, `--max-content-bytes`, `--format json`을 수용하고 성공
   통계 출력한다.
3. `build`: provider/model 선택, env/`--api-key` 처리 범위 확정, logging level과
   `--log-format text|json`, plan service wiring을 한다.
4. `search`: manifest embedding으로 provider/model을 생성하고 options/default를
   적용한다. text 기본은 data만, JSON은 contract field, `--include-content`는
   manifest content 여부 검사, `--include-vector`는 JSON일 때만 허용한다.
5. `inspect`: quick mode만 사용하고 summary/list/id, list+id mutual exclusion,
   text/json/pretty 렌더링을 구현한다. `verify`: full verifier 결과를 요구된
   text/json schema로 출력한다.

수용 테스트:

- subprocess 또는 command-level tests로 각 명령의 required flag, default,
  malformed enum/range, stderr+exit code를 확인한다.
- README/Requirements의 대표 text 및 JSON 결과를 golden으로 비교한다. JSON
  pretty 여부와 content/vector flag 제한을 확인한다.
- search가 manifest model과 다른 provider를 지원하지 않을 때 명확하게 실패하고,
  CLI와 API가 같은 파일·fake vector에서 동점까지 일치한다.

### 단계 7 — evaluator 검토 체크리스트

각 actor PR/작업 단위에서 evaluator는 구현을 다시 작성하지 말고 다음을 검토하고,
발견된 계약 결함에는 먼저 회귀 테스트를 추가한 뒤 actor에게 수정 사항을 전달한다.

- 공개 API의 exported name, errors.Is wrapping, `json.RawMessage` ownership/copy,
  concurrent read-only 안전성, zero-value options를 검토한다.
- 파일 입력이 untrusted라는 전제에서 integer overflow, allocation before bounds
  check, short read, trailing data 정책, checksum 범위를 점검한다.
- API key/Authorization/content가 error, structured/text debug log, `.embed`, temp
  file name에 유출되지 않는지 검색한다.
- retry가 context cancellation을 지연시키지 않고, output atomicity가 OS rename
  failure를 포함해 유지되는지 검토한다.
- README 기본값/예제, Architecture 구조/format, Requirements output/error 기준이
  구현 및 `--reuse` 최종 범위와 일치하는지 검토한다.

### 단계 8 — 품질 게이트 및 후속 산출물

1. file parser/input parser fuzz tests와 1k/10k/50k × 512/1024/1536 benchmark를
   추가한다. benchmark는 CI 필수 pass/fail보다는 성능 기준선 기록용으로 둔다.
2. CI에 `gofmt` 확인, `go test ./...`, `go vet ./...`, `go test -race ./...`,
   `git diff --check`를 넣는다. staticcheck 및 GoReleaser/5개 target/archives/
   checksums는 Requirements의 릴리스 자동화 요구에 맞춰 구성한다.
3. 문서 계약 변경이 있으면 같은 change set에서 README(사용법/default),
   Architecture(구조/API/format), Requirements(behavior/output/test)를 고친다.

최종 수용 명령:

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
go test -race ./...
git diff --check
```

## 4. 명시적으로 후속 버전으로 남길 기능

문서의 MVP 이후 항목인 shell/evaluate, metadata labels filter, batch query CLI,
mmap, quantization, additional providers, HNSW/ANN은 이번 core implementation에
섞지 않는다. 단, evaluator는 README나 CLI가 이를 지원하는 것처럼 표시하지
않는지 확인한다. `--reuse`만은 위 범위 결정에 따라 별도로 처리한다.

## 5. 완료 정의

- 모든 MVP command와 공개 Go API가 문서의 성공/오류/기본값 계약을 만족한다.
- v1 `.embed` writer, quick inspect, full verifier와 loader가 golden/corruption
  테스트에서 상호 호환되고 안전하게 실패한다.
- 실제 network 없이 provider/build/search의 happy/error/retry path가 테스트된다.
- 동시 search race test, 전체 test/vet/format/diff 검사가 통과한다.
- 문서의 `--reuse` 출시 범위 충돌이 최종 구현과 일치하도록 해소돼 있다.
