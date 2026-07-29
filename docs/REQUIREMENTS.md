# embedstore 요구사항

## 제품 개요

`embedstore`는 JSON 콘텐츠를 OpenAI 임베딩으로 변환해 하나의 `.embed` 파일로 만들고, CLI와 Go 모듈에서 메모리 기반 의미 검색을 제공해야 한다.

MVP의 지원 범위는 JSON 배열 입력, OpenAI, `float32`, 정규화, 전체 메모리 로드 및 완전 탐색이다. 기존 `.embed` 병합 build, 대화형 shell, evaluate, metadata 필터, mmap, 양자화 및 ANN 인덱스는 MVP 이후 범위다.

## 기능 요구사항

### 입력 검증

- `embedstore validate --input <path>`는 JSON 문법과 입력 구조를 검사해야 한다.
- 입력은 비어 있지 않은 최상위 `datasetName`, `datasetVersion`과 `items` 배열을 가진 JSON 객체여야 한다.
- `datasetName`은 데이터셋의 안정적인 식별자이고, `datasetVersion`은 해당 데이터셋 원본의 배포 버전이어야 한다.
- 각 항목에는 비어 있지 않은 `content`와 직렬화 가능한 `data`가 있어야 한다. `id`는 선택 사항이다.
- `id`가 제공되면 데이터셋에서 고유해야 하며, 권장 길이 범위는 1~128자다.
- `id`가 없으면 고정된 embedstore UUID namespace와 정규화된 `content`, canonical JSON으로 직렬화한 `data`를 사용해 결정적인 UUID v5 문자열을 생성해야 한다.
- 동일한 content/data 조합으로 생성한 자동 ID 또는 명시적 ID가 중복되면 validation은 실패해야 한다.
- 자동 ID는 content 또는 data 변경 시 달라진다. 변경 후에도 항목 식별자를 유지해야 하는 입력은 명시적 `id`를 제공해야 한다.
- UTF-8, content 최대 바이트 수, 중복 ID, 지원하지 않는 JSON 값과 레코드 수를 검증해야 한다.
- 검증 성공 시 항목 수, 중복 ID 수, 빈 content 수, 잘못된 레코드 수, 예상 토큰 수를 출력해야 한다.
- `--strict`, `--max-content-bytes`, `--format json` 옵션을 제공해야 한다. MVP에서 `--format`의 유효값은 `json`뿐이다.

### 빌드

- `embedstore build`는 `--input`, `--output`, `--embedding`을 받아 `.embed` 파일을 생성해야 한다. `datasetName`과 `datasetVersion`은 입력 JSON에서 읽는다.
- build는 입력 검증, 임베딩 생성, 차원 확인, 정규화, 파일 작성, 파일 검증의 순서로 수행해야 한다.
- 임베딩은 배치 요청으로 순차 전송해야 한다.
- OpenAI 기본 배치 상한은 요청당 최대 100개 항목과 누적 100,000 입력 토큰이어야 한다. 두 상한 중 먼저 도달하는 지점에서 배치를 분할해야 한다.
- MVP의 기본 embedding provider는 OpenAI이며 `OPENAI_API_KEY`를 지원해야 한다.
- `--embedding`은 필수이며 `<provider>/<model>` 형식이어야 한다. MVP의 유효 provider는 `openai`다. 예: `--embedding openai/text-embedding-3-small`.
- `--dimensions`, `--batch-size`, `--max-batch-tokens`, `--overwrite`, `--reuse`, `--include-content`, `--timeout`, `--max-retries` 옵션을 제공해야 한다.
- `--batch-size`는 요청당 최대 항목 수이며 기본값은 100이다. `--max-batch-tokens`는 요청당 누적 입력 토큰 상한이며 기본값은 100,000이다.
- `--timeout`의 기본값은 30초이고, `--max-retries`의 기본값은 5다.
- build의 `--include-content` 기본값은 `true`다. content를 포함한 파일은 이후 `--reuse` 병합 build의 원본으로 사용할 수 있다.
- 저장 벡터와 쿼리 벡터는 항상 L2 정규화해야 하며, 이를 비활성화하는 CLI 옵션을 제공해서는 안 된다.
- 출력 경로에 파일이 이미 있으면 기본적으로 build를 실패시켜야 한다. `--overwrite`를 명시한 경우에만 검증을 마친 임시 파일을 원자적으로 기존 파일과 교체해야 한다.
- `--dimensions`는 선택 사항이며, 파일 벡터를 사후 변환하는 옵션이 아니라 provider/model에 전달하는 임베딩 생성 요청값이어야 한다.
- `--dimensions`를 생략하면 provider의 모델 기본 차원을 사용해야 한다. 지정하면 해당 provider/model이 지원하는 값인지 검증하고 API에 전달해야 한다.
- API 응답의 모든 벡터 길이는 요청한 차원과 일치해야 하며, 미지정인 경우에는 응답 간 차원이 일치해야 한다. 불일치, 지원하지 않는 모델 또는 지원 범위를 벗어난 값은 build를 실패시켜야 한다.
- manifest `dimensions`에는 요청값이 아니라 실제 API 응답 벡터의 차원을 기록해야 한다.
- `--reuse <file.embed>`는 기존 벡터를 재사용하지 않는다. 기존 파일의 content와 metadata를 읽어 입력 JSON의 항목과 ID 기준으로 병합해야 한다.
- 입력 JSON의 `datasetName`과 reuse 파일 manifest의 dataset name이 다르면 build는 데이터셋 불일치 오류를 반환해야 한다. 입력 JSON의 `datasetVersion`은 새 output manifest에 기록한다.
- 병합에서 같은 ID가 있으면 입력 JSON 항목이 우선하고, reuse 파일에만 있는 항목은 결과에 유지해야 한다.
- 병합 결과의 모든 항목은 `--embedding`으로 다시 임베딩해야 한다. reuse 파일과 새 output 파일의 embedding 또는 dimensions가 달라도 된다.
- reuse 파일의 manifest가 `contentIncluded: false`이면 build는 원문 누락 오류를 반환해야 한다.
- 출력은 임시 파일에 먼저 작성한 뒤 검증 성공 시 원자적으로 rename해야 한다. 실패한 build가 손상된 최종 파일을 남겨서는 안 된다.
- OpenAI API key나 민감한 인증 정보는 출력 파일과 로그에 포함되어서는 안 된다.
- content가 외부 임베딩 API로 전송됨을 build 중 경고해야 한다.
- HTTP 429, 500, 502, 503, 504와 일시적 네트워크 오류에는 지수 백오프 재시도를 적용해야 한다.

### 검색

- `embedstore search --file <path> --query <text> --limit <n>`는 쿼리를 임베딩하고 상위 결과를 반환해야 한다.
- `--min-score`, `--api-key`, `--output text|json`, `--include-content`, `--include-vector`, `--pretty` 옵션을 제공해야 한다.
- CLI search는 파일 manifest의 embedding을 읽어 쿼리 임베딩에 사용할 provider/model을 선택해야 한다. manifest에 기록된 provider를 CLI가 지원하지 않으면 명확한 provider 미지원 오류를 반환해야 한다.
- `--limit`의 기본값은 5이며 최대 반환 개수를 뜻한다. `--min-score`를 지정하지 않으면 점수 하한 없이 점수 내림차순의 상위 `limit`개를 반환해야 한다.
- `--min-score`를 지정하면 해당 점수 이상인 결과 중 상위 `limit`개만 반환해야 한다. 따라서 반환 결과 수는 `limit`보다 적거나 0개일 수 있다.
- 결과에는 rank, id, score, content(포함된 경우), 원본 `data`를 포함해야 한다.
- 기본 점수는 정규화된 벡터의 내적으로 계산한 cosine similarity여야 한다.
- 결과는 score 내림차순으로 정렬하고 동점에서는 파일 내 입력 순서, 그 다음 ID로 결정해야 한다.
- Go API에서는 manifest의 embedding/dimensions와 쿼리 embedder가 일치하지 않으면 검색을 거부해야 한다.

#### 검색 출력

기본 출력 형식은 `text`이며, 기본적으로 content와 vector는 포함하지 않는다. 원본 `data`는 모든 출력 형식에서 포함한다.

```bash
embedstore search \
  --file knowledge.embed \
  --query "올해 이직해도 될까요?" \
  --limit 3
```

```text
Query: 올해 이직해도 될까요?
Embedding: openai/text-embedding-3-small
Results: 3

1. score=0.8734 id=career.change.001
   data: {"topic":"career","intent":"job_change"}

2. score=0.8412 id=career.timing.003
   data: {"topic":"career","intent":"job_timing"}

3. score=0.7821 id=career.advice.004
   data: {"topic":"career","intent":"career_advice"}
```

`--include-content`를 지정하면 각 결과에 content 행 또는 `content` JSON 필드를 추가한다. 파일 manifest의 `contentIncluded`가 `false`인 경우에는 content를 반환할 수 없으므로 오류를 반환해야 한다.

```bash
embedstore search \
  --file knowledge.embed \
  --query "올해 이직해도 될까요?" \
  --include-content
```

```text
1. score=0.8734 id=career.change.001
   content: 이직과 직장 이동에 관한 질문
   data: {"topic":"career","intent":"job_change"}
```

`--output json`은 결과를 하나의 JSON 객체로 출력한다. `--pretty`는 JSON 들여쓰기를 적용한다.

```bash
embedstore search \
  --file knowledge.embed \
  --query "올해 이직해도 될까요?" \
  --output json \
  --include-content
```

```json
{
  "query": "올해 이직해도 될까요?",
  "embedding": "openai/text-embedding-3-small",
  "limit": 5,
  "results": [
    {
      "rank": 1,
      "id": "career.change.001",
      "score": 0.8734,
      "content": "이직과 직장 이동에 관한 질문",
      "data": {
        "topic": "career",
        "intent": "job_change"
      }
    }
  ]
}
```

`--include-vector`는 각 결과의 저장된 정규화 벡터를 `vector` JSON 필드로 추가하며, 디버깅 및 검증 용도다. 벡터가 매우 크므로 이 옵션은 `--output json`과 함께만 사용할 수 있어야 한다.

```json
{
  "rank": 1,
  "id": "career.change.001",
  "score": 0.8734,
  "vector": [0.0123, -0.0456, 0.0789]
}
```

### 파일 검사

- `embedstore inspect --file <path>`는 dataset name/version, format version, embedding, dimensions, item count, normalized 여부, vector type, file size, 생성 시각, 기록된 checksum 값을 표시해야 한다. inspect는 전체 checksum 재계산이나 모든 벡터 순회를 수행하지 않는다.
- `inspect`는 `--list`, `--id`, `--output text|json`, `--pretty`를 지원해야 한다. 기본 출력 형식은 `text`다.
- `--list`는 저장 항목의 index와 ID 목록을 출력하고, `--id <id>`는 특정 항목의 index, ID, content(저장된 경우), data를 출력해야 한다.
- `--list`와 `--id`는 동시에 사용할 수 없으며, 함께 지정하면 오류를 반환해야 한다.
- `embedstore verify --file <path>`는 전체 파일 무결성을 검사해야 한다. 검사 항목은 magic bytes, 포맷 버전, 파일 길이와 모든 offset/length 범위, manifest와 metadata 수, 모든 metadata JSON, 벡터 수와 차원, 전체 checksum 재계산, 모든 벡터의 NaN/Inf 및 zero vector다.
- `verify`는 `--output text|json`, `--pretty`를 지원하며 기본 출력 형식은 `text`다.

#### inspect 출력

기본 `inspect`는 파일 manifest의 요약을 출력한다.

```bash
embedstore inspect --file knowledge.embed
```

```text
Dataset name:     app-knowledge
Dataset version:  1.0.0
Format version:   1
Embedding:        openai/text-embedding-3-small
Dimensions:       1536
Items:            5,230
Normalized:       true
Vector type:      float32
Checksum:         sha256:ab12...
```

`--list`는 저장 순서대로 항목 목록을 출력한다.

```bash
embedstore inspect --file knowledge.embed --list
```

```text
0  career.change.001
1  career.timing.003
2  relationship.marriage.001
```

`--id`는 특정 항목을 상세 조회한다.

```bash
embedstore inspect \
  --file knowledge.embed \
  --id career.change.001
```

```text
ID:      career.change.001
Index:   0
Content: 이직과 직장 이동에 관한 질문
Data:    {"topic":"career","intent":"job_change"}
```

`--output json`은 기본, 목록, 상세 조회 결과를 JSON으로 출력하며 `--pretty`로 들여쓰기를 적용할 수 있다.

```bash
embedstore inspect \
  --file knowledge.embed \
  --id career.change.001 \
  --output json \
  --pretty
```

```json
{
  "id": "career.change.001",
  "index": 0,
  "content": "이직과 직장 이동에 관한 질문",
  "data": {
    "topic": "career",
    "intent": "job_change"
  }
}
```

#### verify 출력

`verify`는 배포 전후 또는 CI에서 파일이 손상되지 않았는지 완전하게 확인할 때 사용한다.

```bash
embedstore verify --file knowledge.embed
```

```text
Verification successful

File:       knowledge.embed
Items:      5,230
Vectors:    valid
Metadata:   valid
Checksum:   valid
```

검증 실패 시 명령은 0이 아닌 종료 코드로 끝나며, 오류에는 가능한 경우 파일 path와 byte offset을 포함해야 한다.

```bash
embedstore verify \
  --file knowledge.embed \
  --output json \
  --pretty
```

```json
{
  "file": "knowledge.embed",
  "valid": true,
  "itemCount": 5230,
  "vectors": "valid",
  "metadata": "valid",
  "checksum": "valid"
}
```

### Go 모듈

- 루트 패키지는 `LoadFile`, `NewEngine`, `Search`, `SearchVector`, `Manifest`, `DecodeData`를 제공해야 한다.
- `LoadFile`은 checksum 검증과 memory mode 설정 옵션을 받아야 한다. MVP의 memory mode는 일반 메모리 로드다.
- `Store`는 `Manifest()`, `Count()`, `SearchVector(...)`, `Close()`를 제공해야 한다.
- `Embedder`는 배치 `Embed`, `Embedding`, `Dimensions`를 제공해야 한다.
- `SearchOptions`는 최소한 `Limit`, `MinScore`를 포함해야 한다. Go API에서 `Limit`이 0이면 기본값 5를 적용해야 한다.
- `SearchResult.Data`는 `json.RawMessage`여야 하며 제네릭 decode helper를 제공해야 한다.
- 로드 완료 후 Store는 읽기 전용이며 다수 goroutine에서 동시 검색이 안전해야 한다.
- 라이브러리는 기본적으로 로그를 출력하지 않아야 하며 선택적으로 사용자 제공 logger를 받을 수 있어야 한다.

## 파일 포맷 요구사항

- 파일 확장자는 `.embed`를 사용해야 한다.
- 파일은 magic bytes, file format version, header length, manifest, metadata index, metadata JSON, `float32` vectors, checksum을 포함하는 단일 파일이어야 한다.
- MVP의 vector type은 `float32`만 지원해야 한다.
- 저장 벡터와 쿼리 벡터는 L2 정규화해야 한다.
- manifest에는 format version, dataset name/version, `<provider>/<model>` 형식의 embedding, dimensions, vector type, normalized, item count, created at, source checksum, content 포함 여부를 기록해야 한다.
- 모든 수치 직렬화의 endianness를 문서화하고 writer/reader에서 고정해야 한다.
- reader는 offset, length 및 선언된 count가 파일 크기와 사전 정의된 상한을 넘지 않는지 먼저 확인해야 한다.
- metadata JSON은 원형을 보존하되 deterministic output을 위해 key 정렬을 사용해야 한다.

## 비기능 요구사항

- 동일 입력 순서 및 동일 임베딩 결과에서 파일 구조와 검색 순서는 재현 가능해야 한다. `createdAt`처럼 의도적으로 달라지는 값은 예외다.
- 10,000개 × 1,536차원 `float32` 벡터에서 벡터 데이터 메모리 사용량은 약 60 MiB 수준이어야 한다.
- 검색은 전체 정렬 대신 최소 힙을 써서 Top-K를 유지해야 한다.
- OpenAI 호출은 context 취소, timeout, retry, rate limit, 사용자 정의 HTTP client를 지원해야 한다.
- 로그 레벨 `error`, `warn`, `info`, `debug` 및 `--log-format text|json`을 지원해야 한다.
- API key, 요청 Authorization header, 민감 content를 debug 로그에 노출해서는 안 된다.
- CLI는 Linux amd64/arm64, macOS amd64/arm64, Windows amd64 빌드를 릴리스해야 한다.

## 오류 처리

다음 오류는 `errors.Is`로 판별 가능한 공개 오류여야 한다.

```go
ErrInvalidFile
ErrUnsupportedVersion
ErrDimensionMismatch
ErrModelMismatch
ErrInvalidQuery
```

파일 읽기 오류는 path, byte offset, 원인 오류를 담는 `FileError`로 감싸야 한다. CLI는 사용자가 조치할 수 있는 오류 메시지를 stderr에 출력하고, 실패 시 0이 아닌 종료 코드를 반환해야 한다.

## 품질 검증

- 단위 테스트: 정규화, 내적, Top-K, 입력 검증, manifest, reader/writer, OpenAI client, 공개 API.
- golden file 테스트: 정해진 입력에 대해 reader와 writer의 호환성을 검증.
- 손상 파일 테스트: 잘린 파일, 잘못된 offset, checksum 불일치, metadata 오류, NaN/Inf/zero vector.
- fake HTTP server 테스트: batch ordering, retry, timeout, 오류 분류, context 취소.
- race detector: 동시 `SearchVector` 호출.
- fuzz test: file reader와 입력 JSON parser.
- benchmark: 1천/1만/5만 항목, 512/1,024/1,536 차원, 동시 요청 1/10/100.

## 릴리스 및 단계별 범위

| 버전 | 범위 |
| --- | --- |
| `v0.1.0` | MVP: file format, memory search, OpenAI, validate/build/search/inspect/verify |
| `v0.2.0` | 기존 `.embed` 병합 build, shell, evaluate |
| `v0.3.0` | labels 기반 필터, batch query, benchmark command |
| `v0.4.0` | mmap, 추가 embedding provider |
| `v1.0.0` | 파일 포맷과 공개 API 안정화, 운영 검증 완료 |

릴리스 자동화는 Go test, vet, staticcheck, race test, GoReleaser, 플랫폼별 압축 산출물과 `checksums.txt` 생성을 포함해야 한다.
