# embedstore

`embedstore`는 JSON 콘텐츠를 임베딩 벡터로 변환해 하나의 `.embed` 파일에 저장하고, 이를 메모리에서 빠르게 의미 검색하는 Go 라이브러리와 CLI입니다.

```text
knowledge.json
  -> embedstore build
  -> knowledge.embed
  -> embedstore search 또는 Go 애플리케이션
  -> metadata와 유사도 점수 반환
```

초기 버전은 OpenAI 임베딩, `float32` 벡터, 전체 메모리 로드, cosine similarity 기반 완전 탐색을 제공합니다. CLI와 Go 모듈은 같은 파일 로더와 검색 엔진을 사용합니다.

## 설치

Go 모듈과 CLI는 같은 repository에서 배포됩니다.

```bash
go install github.com/<organization>/embedstore/cmd/embedstore@latest
```

OpenAI를 사용할 경우 API key를 설정합니다.

```bash
export OPENAI_API_KEY="..."
```

테스트 또는 OpenAI-compatible endpoint를 사용할 때는 선택적으로 `OPENAI_BASE_URL`을 설정할 수 있습니다. 기본값은 `https://api.openai.com/v1`입니다.

## 빠른 시작

### 1. 입력 JSON 작성

입력은 데이터셋 정보와 `items` 배열을 가진 JSON 객체입니다.

```json
{
  "datasetName": "app-knowledge",
  "datasetVersion": "1.0.0",
  "items": [
    {
      "id": "career.change.001",
      "content": "이직과 직장 이동에 관한 질문",
      "data": {
        "topic": "career",
        "intent": "job_change"
      }
    },
    {
      "content": "결혼 시기와 배우자 인연에 관한 질문",
      "data": {
        "topic": "relationship",
        "intent": "marriage"
      }
    }
  ]
}
```

`datasetName`, `datasetVersion`, `content`, `data`는 필수입니다. `id`는 선택 사항이며, 없으면 정규화된 `content`와 canonical JSON `data`를 바탕으로 결정적인 UUID v5를 생성합니다. 따라서 같은 입력은 같은 자동 ID를 얻습니다.

content 또는 data를 수정해도 같은 항목으로 취급해야 한다면 명시적 `id`를 사용하세요.

### 2. 검증 및 빌드

```bash
embedstore validate --input knowledge.json

embedstore build \
  --input knowledge.json \
  --output knowledge.embed \
  --embedding openai/text-embedding-3-small
```

build는 입력 검증, OpenAI 임베딩 생성, L2 정규화, 파일 작성, 파일 검증을 수행한 뒤 성공 시에만 최종 파일을 원자적으로 만듭니다.

### 3. 검색

```bash
embedstore search \
  --file knowledge.embed \
  --query "올해 이직해도 될까요?"
```

`search`는 파일 manifest의 `embedding`을 읽어 쿼리 임베딩에 사용할 provider/model을 자동으로 선택합니다.

```text
Query: 올해 이직해도 될까요?
Embedding: openai/text-embedding-3-small
Results: 2

1. score=0.8734 id=career.change.001
   data: {"topic":"career","intent":"job_change"}

2. score=0.8412 id=relationship.marriage.001
   data: {"topic":"relationship","intent":"marriage"}
```

### 4. 파일 확인

```bash
embedstore inspect --file knowledge.embed
embedstore verify --file knowledge.embed
```

`inspect`는 빠른 manifest 조회이고, `verify`는 checksum·metadata·모든 벡터를 검사하는 전체 무결성 확인입니다.

## CLI 사용법

### `validate`

입력 JSON의 구조, 필수 필드, 중복 ID, UTF-8, content 크기를 검사합니다.

```bash
embedstore validate --input knowledge.json --strict
```

| 파라미터 | 기본값 | 설명 |
| --- | --- | --- |
| `--input <path>` | 필수 | 입력 JSON 경로 |
| `--strict` | `false` | 더 엄격한 입력 검증 적용 |
| `--max-content-bytes <n>` | 미설정 | 항목별 content 최대 바이트 수 |

### `build`

JSON을 `.embed` 파일로 빌드합니다. 원본 content는 OpenAI API로 전송됩니다.

```bash
embedstore build \
  --input knowledge.json \
  --output dist/knowledge.embed \
  --embedding openai/text-embedding-3-small
```

| 파라미터 | 기본값 | 설명 |
| --- | --- | --- |
| `--input <path>` | 필수 | 입력 JSON 경로 |
| `--output <path>` | 필수 | 생성할 `.embed` 파일 경로 |
| `--embedding <provider>/<model>` | 필수 | 임베딩 식별자. MVP는 `openai` provider 지원 |
| `--dimensions <n>` | 모델 기본값 | provider에 요청할 출력 벡터 차원. 실제 API 응답 차원이 manifest에 기록됨 |
| `--batch-size <n>` | `100` | 요청 하나에 넣을 최대 항목 수 |
| `--max-batch-tokens <n>` | `100000` | 요청 하나의 누적 **추정** 입력 토큰 상한. `0` 이하는 토큰 상한 비활성화 |
| `--timeout <duration>` | `30s` | OpenAI 요청 timeout |
| `--max-retries <n>` | `5` | 일시적 API/네트워크 오류의 최대 재시도 횟수(최초 요청 제외) |
| `--include-content=<bool>` | `true` | output 파일에 원본 content 저장 여부 |
| `--reuse <path>` | 미설정 | 병합할 기존 `.embed` 파일 |
| `--overwrite` | `false` | 기존 output 파일의 원자적 교체 허용 |

배치는 순차적으로 처리합니다. 항목 수(`batch-size`) 또는 content의 보수적 추정 토큰 수(`max-batch-tokens`) 중 먼저 상한에 도달하면 다음 요청으로 분할합니다. `--max-batch-tokens=0`은 토큰 상한을 비활성화합니다.

`dimensions`는 파일 벡터를 사후 변환하는 기능이 아닙니다. 지원되는 모델과 값일 때만 provider에 전달되며, 미설정 시 모델 기본 차원이 사용됩니다.

#### 기존 `.embed` 병합

`--reuse`는 기존 벡터를 복사하는 기능이 아닙니다. 기존 파일의 content와 metadata를 새 입력 JSON과 ID 기준으로 병합한 뒤, 병합된 모든 항목을 `--embedding`으로 다시 임베딩합니다.

```bash
embedstore build \
  --input knowledge-update.json \
  --reuse knowledge-v1.embed \
  --output knowledge-v2.embed \
  --embedding openai/text-embedding-3-small
```

- 입력 JSON의 `datasetName`은 reuse 파일의 dataset name과 같아야 합니다.
- 같은 ID가 양쪽에 있으면 입력 JSON 항목이 우선합니다.
- reuse 파일에만 있는 항목은 결과에 유지됩니다.
- reuse 파일은 `contentIncluded: true`여야 합니다.
- 이전 파일과 새 파일의 embedding 또는 dimensions는 달라도 됩니다.

### `search`

파일을 로드하고 쿼리를 임베딩해 유사한 항목을 찾습니다. 기본적으로 상위 5개를 text로 출력하며 content와 vector는 숨깁니다.

```bash
embedstore search \
  --file knowledge.embed \
  --query "올해 이직해도 될까요?" \
  --limit 10 \
  --min-score 0.65 \
  --include-content
```

| 파라미터 | 기본값 | 설명 |
| --- | --- | --- |
| `--file <path>` | 필수 | 검색할 `.embed` 파일 |
| `--query <text>` | 필수 | 검색 문장 |
| `--limit <n>` | `5` | 최대 반환 결과 수 |
| `--min-score <n>` | 미설정 | 반환할 최소 cosine similarity 점수 |
| `--api-key <key>` | `OPENAI_API_KEY` | 쿼리 임베딩용 OpenAI API key |
| `--output text|json` | `text` | 출력 형식 |
| `--pretty` | `false` | JSON 들여쓰기 |
| `--include-content` | `false` | 결과에 저장된 content 포함 |
| `--include-vector` | `false` | 결과에 저장된 정규화 벡터 포함. `json` 출력에서만 사용 |

`min-score`를 생략하면 점수 하한 없이 높은 점수부터 `limit`개를 반환합니다. 지정하면 기준 이상인 결과만 반환하므로 결과 수는 `limit`보다 적거나 0개일 수 있습니다.

```bash
embedstore search \
  --file knowledge.embed \
  --query "올해 이직해도 될까요?" \
  --output json \
  --include-content \
  --pretty
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

`--include-content`는 파일이 content를 저장한 경우에만 사용할 수 있습니다. `--include-vector`는 큰 응답을 만들 수 있으므로 디버깅과 검증 용도로만 사용하세요.

### `inspect`

파일 manifest 또는 저장 항목을 빠르게 조회합니다. checksum을 재계산하거나 전체 벡터를 순회하지는 않습니다.

```bash
embedstore inspect --file knowledge.embed
embedstore inspect --file knowledge.embed --list
embedstore inspect --file knowledge.embed --id career.change.001
```

| 파라미터 | 기본값 | 설명 |
| --- | --- | --- |
| `--file <path>` | 필수 | 조회할 `.embed` 파일 |
| `--list` | `false` | 저장 순서의 index와 ID 목록 출력 |
| `--id <id>` | 미설정 | 특정 항목의 index, ID, content, data 조회 |
| `--output text|json` | `text` | 출력 형식 |
| `--pretty` | `false` | JSON 들여쓰기 |

`--list`와 `--id`는 함께 사용할 수 없습니다.

### `verify`

파일 전체의 무결성을 검사합니다.

```bash
embedstore verify --file knowledge.embed
```

| 파라미터 | 기본값 | 설명 |
| --- | --- | --- |
| `--file <path>` | 필수 | 검사할 `.embed` 파일 |
| `--output text|json` | `text` | 출력 형식 |
| `--pretty` | `false` | JSON 들여쓰기 |

검사 범위에는 magic bytes, 포맷 버전, 모든 offset/length, metadata JSON, 벡터 수와 차원, checksum 재계산, NaN/Inf 및 zero vector가 포함됩니다.

## Go 모듈 API

### 설치

```bash
go get github.com/<organization>/embedstore
```

### 문자열 검색

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
    "os"

    "github.com/<organization>/embedstore"
    openaiembedding "github.com/<organization>/embedstore/embedding/openai"
)

type Metadata struct {
    Topic  string `json:"topic"`
    Intent string `json:"intent"`
}

func main() {
    ctx := context.Background()

    store, err := embedstore.LoadFile("knowledge.embed")

    if err != nil {
        log.Fatal(err)
    }
    defer store.Close()

    embedder := openaiembedding.New(
        openaiembedding.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )

    engine, err := embedstore.NewEngine(store, embedder)
    if err != nil {
        log.Fatal(err)
    }

    results, err := engine.Search(ctx, "올해 이직해도 될까요?", embedstore.SearchOptions{
        Limit:    10,
        MinScore: 0.65,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, result := range results {
        var data Metadata
        if err := json.Unmarshal(result.Data, &data); err != nil {
            log.Fatal(err)
        }
        fmt.Printf("%d. %.4f %s (%s)\n", result.Rank, result.Score, result.ID, data.Topic)
    }
}
```

기본 `LoadFile(path)`은 checksum을 항상 검증합니다. 신뢰할 수 있는 로컬 진단에서만 `LoadFileWithOptions(path, LoadOptions{VerifyChecksum: &disabled})`처럼 checksum 검사만 생략할 수 있으며, 이 경우에도 파일 구조와 metadata/vector 검증은 그대로 수행됩니다.

`NewEngine`은 store manifest의 embedding/dimensions와 embedder가 호환되는지 기본적으로 검사합니다. provider 또는 모델이 다르면 `ErrModelMismatch`를 반환합니다.

### 이미 가진 벡터로 검색

```go
results, err := store.SearchVector(ctx, queryVector, embedstore.SearchOptions{
    Limit: 5,
})
```

`queryVector`는 파일 manifest와 같은 차원이어야 합니다. 검색 전에 정규화되며, 결과는 score 내림차순으로 반환됩니다.

### 주요 API와 파라미터

| API | 주요 파라미터 | 설명 |
| --- | --- | --- |
| `LoadFile(path)` | `path`: `.embed` 경로 | checksum을 검증하며 파일을 메모리에 로드 |
| `LoadFileWithOptions(path, options)` | `path`, `LoadOptions` | 명시적 loader 옵션으로 파일을 메모리에 로드 |
| `LoadOptions` | `VerifyChecksum *bool`, `MemoryMode` | checksum은 nil/기본값에서 검증하며, `MemoryModeLoad`만 MVP에서 지원 |
| `NewEngine(store, embedder)` | `store`, `embedder` | 문자열 검색용 Engine 생성 및 호환성 검사 |
| `Engine.Search(ctx, query, options)` | `query`: 검색 문자열 | embedder로 쿼리를 임베딩한 뒤 검색 |
| `Store.SearchVector(ctx, vector, options)` | `vector`: 쿼리 벡터 | 이미 임베딩된 벡터로 검색 |
| `SearchOptions.Limit` | 기본 `5` (`0`일 때) | 최대 반환 결과 수 |
| `SearchOptions.MinScore` | 기본 없음 | 최소 cosine similarity 점수 |
| `DecodeData[T](result)` | `result`: `SearchResult` | `Data json.RawMessage`를 타입 `T`로 변환 |

`Store`는 로드 후 읽기 전용이므로 여러 goroutine에서 안전하게 동시에 검색할 수 있습니다.

## 파일과 보안

- `.embed` 파일에는 API key를 저장하지 않습니다.
- `contentIncluded: true` 파일에는 원문 content가 포함됩니다. 파일 접근 권한과 배포 범위를 관리하세요.
- build는 content를 외부 provider에 전송합니다. 민감한 원본 데이터를 사용할 때는 조직의 보안·보존 정책을 먼저 확인하세요.
- build 실패 시 기존 output 파일은 바꾸지 않습니다. 기존 파일을 교체하려면 `--overwrite`를 명시하세요.

## CI와 릴리스

GitHub Actions CI는 모든 push와 pull request에서 포맷, 테스트, vet, diff, staticcheck 및 Linux race detector를 검사합니다. `v*` 태그를 push하면 GoReleaser가 Linux amd64/arm64, macOS amd64/arm64, Windows amd64용 archive와 SHA-256 `checksums.txt`를 포함한 GitHub Release를 만듭니다.

```bash
git tag v0.1.0
git push origin v0.1.0
```

태그를 만들기 전에 일반 push 또는 pull request CI가 통과했는지 확인하세요.

## 개발: 빌드와 테스트

Go 1.22 이상에서 다음 명령으로 라이브러리와 CLI를 빌드할 수 있습니다.

```bash
go build ./...
go build -o ./bin/embedstore ./cmd/embedstore
./bin/embedstore
```

변경 후에는 다음 기본 검증을 실행합니다.

```bash
go test ./...
go vet ./...
git diff --check
```

동시성, 파일 reader, 공개 API 또는 검색 엔진을 변경한 경우에는 지원되는 환경에서 race detector도 실행합니다.

```bash
go test -race ./...
```

## 문서

- [아키텍처](docs/ARCHITECTURE.md)
- [요구사항](docs/REQUIREMENTS.md)
