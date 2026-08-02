# embedstore 아키텍처

`embedstore`는 도메인에 독립적인 JSON 콘텐츠를 OpenAI 임베딩으로 변환해 단일 `.embed` 파일에 저장하고, 이를 메모리에 적재하여 유사도 검색하는 Go 라이브러리와 CLI이다.

## 목표와 경계

동일한 파일 로더와 검색 엔진을 CLI와 애플리케이션이 함께 사용한다. 따라서 개발 환경의 `embedstore search` 결과와 애플리케이션의 검색 결과는 같은 입력 벡터 및 옵션에서 결정적으로 일치해야 한다.

```text
source.json
  -> embedstore CLI build
  -> knowledge.embed
  -> CLI search / Go module
  -> query embedding
  -> in-memory similarity search
  -> metadata and score
```

프로젝트 식별자는 다음으로 통일한다.

| 대상 | 이름 |
| --- | --- |
| 프로젝트 / 패키지 | `embedstore` |
| CLI 바이너리 | `embedstore` |
| Go module | `github.com/gamzabox/embedstore` |
| 벡터 파일 | `*.embed` |

초기 버전은 수천~수만 개 항목을 대상으로 한 완전 탐색을 지원한다. HNSW, 메모리 매핑, 양자화, 다중 provider는 확장 범위다.

## 구성 요소

```text
cmd/embedstore       CLI 진입점
internal/cli         build, validate, search, inspect, verify 명령
embedding            Embedder 인터페이스와 provider 구현
embedding/openai     OpenAI Embeddings API 클라이언트
internal/fileformat  .embed writer, reader, checksum 및 검증
internal/heap        Top-K 최소 힙
internal/normalize   float32 벡터 정규화

root package         Store, MemoryStore, Engine, 공개 타입과 오류
```

공개 API는 루트 패키지에 집중한다. 파일 포맷과 CLI 구현 세부 사항은 `internal`에 둔다.

## 데이터 흐름

### 빌드

1. CLI가 JSON 배열 입력을 읽고 스키마와 중복 ID를 검증한다.
2. `--reuse`가 지정되면 기존 `.embed`에서 content와 metadata를 읽어 입력 JSON의 항목과 병합한다.
3. 병합된 모든 `content`를 보수적인 크기의 배치로 나누어 OpenAI에 순차적으로 전달해 임베딩을 만든다.
4. 응답 차원을 검증하고 모든 벡터를 L2 정규화한다.
5. metadata와 연속된 `float32` 벡터 배열을 임시 파일에 기록한다.
6. 생성 파일을 다시 검증하고 checksum을 확인한다.
7. 성공한 경우에만 원자적 rename으로 최종 `.embed` 파일을 만든다.

모든 벡터와 쿼리 벡터는 항상 L2 정규화한다.

기본적으로 출력 경로에 파일이 이미 있으면 build는 실패한다. `--overwrite`를 명시한 경우에만 검증까지 완료한 임시 파일을 원자적으로 교체한다. 실패 시 최종 출력 파일은 만들거나 덮어쓰지 않는다. 임시 파일은 `output.embed.tmp`와 같이 출력 파일과 같은 디렉터리에 만든다.

### 검색

1. `LoadFile(path)`은 manifest, metadata, 벡터를 검증한 후 메모리에 적재한다. `LoadFileWithOptions(path, options)`는 명시적 loader 옵션이 필요한 경우에 사용한다.
   기본 호출은 checksum 검증을 수행한다. `LoadOptions.VerifyChecksum`이 nil이면 검증하며, false는 checksum 비교만 생략한다. MVP는 빈 memory mode 또는 `MemoryModeLoad`만 허용하고 다른 mode는 오류다. `VerifyFile`은 옵션과 무관하게 항상 checksum을 검증한다.
2. CLI search는 manifest의 embedding으로 provider client를 선택하고 문자열 쿼리를 임베딩한다. Go `Engine.Search`는 제공받은 `Embedder`로 쿼리를 임베딩한다.
3. 쿼리 벡터를 정규화하고 파일 manifest의 임베딩 식별자 및 차원과 호환되는지 확인한다.
4. `MemoryStore.SearchVector`가 모든 저장 벡터와 내적을 계산한다.
5. 최소 힙으로 Top-K만 유지하고, 점수 내림차순으로 결과를 반환한다.

정규화된 벡터의 내적은 cosine similarity와 같다. 동점은 파일 내 입력 순서, 그 다음 ID 오름차순으로 정해 결과를 항상 결정적으로 만든다.

검색의 기본 `Limit`은 5이며, Go API에서 `SearchOptions.Limit`이 0인 경우에도 같은 기본값을 적용한다. `MinScore`를 지정하지 않으면 점수 하한 없이 상위 `Limit`개를 반환하고, 지정하면 해당 점수 이상인 결과만 반환한다.

## 입력 모델

MVP는 JSON 배열 래퍼를 지원한다.

```json
{
  "datasetName": "app-knowledge",
  "datasetVersion": "1.0.0",
  "items": [
    {
      "id": "career.change.001",
      "content": "이직과 직장 이동에 관한 질문",
      "data": {"topic": "career", "intent": "job_change"}
    }
  ]
}
```

최상위 `datasetName`, `datasetVersion`과 각 항목의 `content`, `data`가 필수다. `id`는 선택 사항이다. `datasetName`은 안정적인 데이터셋 식별자이고, `datasetVersion`은 해당 원본 데이터의 배포 버전이다. build는 이 값을 manifest에 그대로 기록한다. `data`는 임의 JSON을 보존하며 결과에서는 `json.RawMessage`로 반환한다.

항목에 `id`가 있으면 그 값을 그대로 사용한다. 없으면 고정된 embedstore UUID namespace와 정규화한 `content`, canonical JSON으로 직렬화한 `data`를 이름으로 사용해 UUID v5를 생성한다. 이 방식은 UUID 문자열 형식과 재현성을 모두 보장한다.

```text
id = UUIDv5(embedstore-fixed-namespace, normalized(content) + canonical-json(data))
```

content 또는 data가 달라지면 자동 생성 ID도 달라진다. 같은 content/data 조합은 같은 ID를 생성하므로 validation에서 중복 ID 오류가 된다. 내용이 바뀌어도 ID를 유지하거나 외부 시스템과 항목을 연결해야 하는 경우에는 명시적 `id`를 제공해야 한다.

데이터셋 이름과 버전은 입력 JSON이 소유하므로 옵션 누락으로 다른 데이터셋 manifest가 만들어지는 일을 방지한다. CI는 입력 JSON을 생성하거나 갱신할 때 Git tag 또는 commit SHA 등을 `datasetVersion`에 기록할 수 있다. 선택 필드(`tags`, `enabled`, `weight`, `labels`)는 포맷 호환성을 고려해 후속 버전에서 추가한다.

## 기존 `.embed` 병합 빌드

`build --reuse <file.embed>`의 `reuse`는 기존 벡터의 재사용이 아니라 기존 파일에 포함된 content와 metadata의 재사용을 뜻한다.

```bash
embedstore build \
  --input knowledge-update.json \
  --reuse knowledge-v1.embed \
  --output knowledge-v2.embed \
  --embedding openai/text-embedding-3-small
```

build는 입력 JSON의 항목과 reuse 파일의 항목을 ID 기준으로 병합한다. 입력 JSON의 `datasetName`은 reuse 파일 manifest의 dataset name과 같아야 한다. 같은 ID가 양쪽에 있으면 입력 JSON 항목이 우선하며, reuse 파일에만 있는 항목은 결과에 유지한다. 따라서 입력 JSON에는 새 항목과 변경할 항목만 넣어도 된다. 자동 UUID v5 ID는 content 또는 data 변경 시 달라지므로, 기존 항목을 대체하려면 명시적 `id`를 사용해야 한다.

병합된 결과의 모든 항목은 `--embedding`으로 다시 임베딩한다. 기존 벡터는 읽거나 복사하지 않으며, 이전 파일의 임베딩 식별자 및 차원과 새 값이 달라도 된다. output manifest에는 입력 JSON의 dataset name/version과 새 임베딩 식별자 및 차원을 기록한다. reuse 파일은 `contentIncluded: true`여야 하며, 그렇지 않으면 원문이 없으므로 build는 오류를 반환한다.

## `.embed` 파일 포맷

파일은 배포와 복사를 쉽게 하는 단일 바이너리다. 정수와 float32의 byte order는 명시적으로 고정하며, 모든 offset 및 length는 파일 크기 범위 안인지 검증한다.

```text
magic bytes | format version | header length
manifest
sequential item records (metadata length + metadata JSON)
contiguous float32 vectors
checksum
```

Version 1 uses little-endian encoding for fixed-width fields. After the four-byte `EMBD` magic, the header is a `uint16` format version and a `uint32` manifest length followed by manifest JSON. It then stores exactly `itemCount` sequential item records, each a little-endian `uint32` metadata length followed by that item JSON; v1 has no metadata index. The vector region follows immediately and contains `itemCount * dimensions` little-endian IEEE-754 `float32` values in item order. The final 32 bytes are the SHA-256 checksum of every preceding byte.


Manifest에는 최소한 다음을 기록한다.

```json
{
  "formatVersion": 1,
  "datasetName": "knowledge",
  "datasetVersion": "1.0.0",
  "embedding": "openai/text-embedding-3-small",
  "dimensions": 1536,
  "vectorType": "float32",
  "normalized": true,
  "itemCount": 5230,
  "createdAt": "2026-07-29T12:30:00Z",
  "sourceChecksum": "sha256:...",
  "contentIncluded": true
}
```

체크섬은 magic부터 checksum 직전까지의 모든 바이트를 대상으로 한다. reader는 magic, 지원 포맷 버전, 파일 길이, 항목 수, metadata, 벡터 차원, NaN/Inf, zero vector 및 checksum을 검증한다.

CLI의 `inspect`와 `verify`는 검증 깊이가 다르다. `inspect`는 manifest와 저장된 checksum 값을 빠르게 조회하며 checksum 재계산이나 전체 벡터 순회는 하지 않는다. `verify`는 모든 offset/length, metadata JSON, 벡터 및 checksum을 전체 검사해 파일 무결성을 확인한다.

## 메모리와 검색 엔진

`MemoryStore`는 벡터를 `[][]float32`가 아닌 단일 연속 배열로 보관한다.

```go
type MemoryStore struct {
    manifest Manifest
    items    []Item
    vectors  []float32 // index * dimensions : (index + 1) * dimensions
}
```

이 방식은 GC 객체 수를 줄이고 파일 레이아웃과 직접 대응한다. Store는 로드 완료 후 읽기 전용이므로 여러 goroutine이 락 없이 동시에 검색할 수 있다.

검색 복잡도는 벡터 비교에 `O(N * D)`, Top-K 유지에 `O(N log K)`다. 10,000개 × 1,536차원 `float32` 벡터의 원시 벡터 메모리는 약 60 MiB다.

## 공개 Go API

```go
type Store interface {
    Manifest() Manifest
    Count() int
    SearchVector(context.Context, []float32, SearchOptions) ([]SearchResult, error)
    Close() error
}

type Embedder interface {
    Embed(context.Context, []string) ([][]float32, error)
    Embedding() string
    Dimensions() int
}
```

```go
store, err := embedstore.LoadFile("knowledge.embed")
embedder := openaiembedding.New(openaiembedding.WithAPIKey(apiKey))
engine, err := embedstore.NewEngine(store, embedder)
results, err := engine.Search(ctx, "올해 이직해도 될까요?", embedstore.SearchOptions{Limit: 10})
```

`SearchResult`는 `Rank`, `ID`, `Score`, `Content`, `Data json.RawMessage`를 가진다. `DecodeData[T]` 헬퍼는 사용자가 자신의 metadata 타입으로 변환할 수 있게 한다.

임베딩 식별자는 `<provider>/<model>` 형식의 정규화된 문자열을 사용한다. 예를 들어 `openai/text-embedding-3-small`이다. 기본적으로 `NewEngine`은 파일 manifest의 embedding과 dimensions가 `Embedder`와 일치하는지 검사한다. 같은 차원이라도 provider 또는 모델이 다르면 벡터 공간이 호환되지 않으므로 `ErrModelMismatch`를 반환한다. 특별한 custom embedder에는 호환성 검사를 끄는 옵션을 제공할 수 있으나 기본값은 검사 활성화다.

CLI search는 별도의 임베딩 선택을 받지 않고 파일 manifest의 embedding으로 provider와 모델을 선택한다. 반면 Go API는 애플리케이션이 `Embedder`를 주입하며, `NewEngine`의 호환성 검사가 해당 embedder와 파일 manifest의 일치를 보장한다.

## OpenAI 연동

OpenAI 구현은 `embedding/openai` 패키지에 둔다. API key는 환경변수 또는 옵션으로만 받고 파일과 로그에 절대 저장하지 않는다. 클라이언트는 배치 요청, context 취소, timeout, 사용자 HTTP client, 응답 순서 보장 및 차원 검증을 지원한다.

OpenAI MVP의 기본 배치 정책은 요청당 최대 100개 항목 및 누적 100,000 입력 토큰이다. 배치는 두 상한 중 먼저 도달하는 지점에서 분할하며, 요청은 순차적으로 처리한다. 이는 rate limit, timeout 및 재시도 범위를 보수적으로 제한하기 위한 정책이다.

`dimensions`는 파일의 벡터를 임의로 변환하는 설정이 아니라 provider/model에 전달하는 임베딩 생성 요청값이다. `--dimensions`를 생략하면 provider의 모델 기본 차원을 사용한다. 지정한 경우 OpenAI client는 이를 API에 전달하고, 반환된 모든 벡터의 길이가 요청값과 일치하는지 검증한다. 차원을 지원하지 않는 모델 또는 지원 범위를 벗어난 값은 build 오류가 된다. manifest에는 요청값이 아니라 실제 API 응답 벡터의 차원을 기록한다.

429와 일시적 5xx(500, 502, 503, 504), 일시적 네트워크 오류는 지수 백오프로 재시도한다. 영구적인 4xx와 입력 오류는 즉시 반환한다.

## 오류 및 보안 원칙

공개 sentinel 오류는 `ErrInvalidFile`, `ErrUnsupportedVersion`, `ErrDimensionMismatch`, `ErrModelMismatch`, `ErrInvalidQuery`를 제공한다. 파일 관련 상세 정보는 path와 offset을 포함하는 `FileError`로 감싼다.

입력 및 파일 reader는 공격적 또는 손상된 파일이 과도한 메모리를 할당하지 못하게 최대 크기와 offset/length를 먼저 검사한다. 원본 content는 외부 임베딩 API로 전송될 수 있으므로 CLI 문서와 build 로그에서 이를 명확히 고지한다.

## 버전 정책과 확장

Go module 버전, dataset 버전, file format 버전은 각각 독립적이다. 예를 들어 module `v0.3.0`이 file format `1`, dataset `1.7.0`을 읽을 수 있다.

MVP 후에는 shell, 평가 도구, 평면 labels 필터, mmap, 양자화, 다른 provider, HNSW를 차례로 추가한다. 기존 `.embed` 병합 빌드는 v0.1 MVP에 포함한다. 파일 포맷 버전 1은 v1.0.0 이전에도 호환성 정책을 명확히 유지하며, 호환되지 않는 변경은 새 포맷 버전으로만 도입한다.
