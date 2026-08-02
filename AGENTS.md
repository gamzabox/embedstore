# embedstore 개발 가이드

## 프로젝트 개요

`embedstore`는 JSON 콘텐츠를 임베딩 벡터로 변환해 단일 `.embed` 파일에 저장하고, 메모리 기반 의미 검색을 제공하는 Go 라이브러리와 CLI다.

구현의 기준은 다음 설계 문서다. 기능을 추가하거나 변경하기 전에 관련 문서를 읽고, 구현으로 인해 계약이 달라지면 같은 변경에서 문서도 갱신한다.

| 문서 | 용도 |
| --- | --- |
| [README.md](README.md) | 사용자 설치, CLI 사용법, 기본값, Go API 예제 |
| [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) | 시스템 경계, 데이터 흐름, 파일 포맷, 검색 엔진, 공개 API 설계 |
| [docs/REQUIREMENTS.md](docs/REQUIREMENTS.md) | 기능 요구사항, CLI 출력 계약, 기본값, 검증·테스트·릴리스 요구사항 |

문서 간 역할을 지킨다.

- 사용자 사용법, 예제, 파라미터 기본값은 `README.md`에 둔다.
- 구현 구조, 데이터 흐름, 포맷 또는 공개 API 설계 변경은 `docs/ARCHITECTURE.md`를 갱신한다.
- 기능 동작, CLI 출력 계약, 오류·검증·테스트 기준 변경은 `docs/REQUIREMENTS.md`를 갱신한다.
- 새 문서가 필요한 경우 `docs/`에 Markdown으로 추가하고 README의 문서 목록에도 연결한다.

## 구현 원칙

- CLI와 Go 라이브러리는 같은 파일 로더와 검색 엔진을 사용한다.
- `.embed` 파일 포맷은 공개 계약이다. 호환되지 않는 변경은 기존 포맷을 수정하지 말고 새 file format version으로 도입한다.
- API key, Authorization header, 민감 content를 로그·오류 메시지·`.embed` 파일에 기록하지 않는다.
- build는 임시 파일 작성과 검증 후 원자적 rename을 사용한다. 실패한 build가 기존 정상 output 파일을 손상시키면 안 된다.
- 공개 Go API의 변경은 하위 호환성, sentinel 오류의 `errors.Is` 동작, 예제 코드를 함께 검토한다.
- 기본값 또는 CLI 파라미터를 변경하면 README와 요구사항 문서를 모두 확인한다.
- 사용자가 요청하지 않은 대규모 리팩터링, 파일 포맷 변경, 의존성 교체는 하지 않는다.

## TDD와 테스트

모든 동작 변경은 테스트 주도로 구현한다.

1. 요구사항과 수용 기준을 확인한다.
2. 실패하는 단위 테스트 또는 회귀 테스트를 먼저 작성한다.
3. 테스트를 통과하는 최소 구현을 작성한다.
4. 중복 제거와 가독성 개선을 리팩터링한 뒤 테스트를 다시 실행한다.
5. 오류 경로와 경계값도 테스트한다.

다음은 최소 검증 기준이다.

```bash
gofmt -w <changed-go-files>
go test ./...
go vet ./...
```

동시성, 파일 reader, 공개 API 또는 검색 엔진을 변경하면 아래도 실행한다.

```bash
go test -race ./...
```

테스트가 실패하면 실패를 숨기거나 우회하지 않는다. 원인을 해결한 뒤 관련 테스트가 모두 통과했음을 확인한다. 실행하지 못한 검증이 있으면 이유와 영향을 명확히 보고한다.

## 기능별 확인 사항

### 파일 포맷 및 loader

- magic bytes, format version, 파일 길이, 모든 offset/length, count 상한을 검증한다.
- checksum, metadata JSON, vector count/dimensions, NaN/Inf/zero vector의 성공·실패 사례를 테스트한다.
- reader/writer 변경에는 golden file 및 손상 파일 테스트를 추가한다.

### 임베딩과 build

- embedding은 `<provider>/<model>` 식별자를 사용한다.
- `dimensions`는 provider 요청값이며, 실제 API 응답 차원을 검증하고 manifest에 기록한다.
- 배치 기본값과 순차 처리 정책을 유지한다.
- 재시도, timeout, context 취소, 응답 순서, API key 비노출을 테스트한다.
- 단위 테스트는 실제 OpenAI API를 호출하지 않는다. `httptest` 또는 fake `Embedder`로 성공·실패 응답을 재현한다.

### 검색

- 저장·쿼리 벡터를 L2 정규화하고 cosine similarity를 내적으로 계산한다.
- Top-K, 동점 정렬, `Limit` 기본값, `MinScore`, 차원 불일치를 테스트한다.
- Store의 동시 검색 안전성을 race detector로 확인한다.
- CLI search는 파일 manifest의 embedding으로 provider/model을 선택한다.

### CLI와 문서

- CLI 옵션, 기본값, text/JSON 출력은 `README.md` 및 `docs/REQUIREMENTS.md`와 일치해야 한다.
- `inspect`는 빠른 manifest 조회, `verify`는 전체 파일 검증이라는 역할 분리를 유지한다.
- 사용자에게 보이는 오류는 조치 가능한 메시지와 0이 아닌 종료 코드를 제공한다.

## 작업 완료 전 점검

- 변경 범위에 맞는 테스트가 추가되었는가?
- `go test ./...`가 통과했는가?
- 필요한 경우 `go vet ./...` 및 `go test -race ./...`가 통과했는가?
- 변경한 Go 파일에 `gofmt`를 적용했는가?
- `git diff --check`가 통과했는가?
- 사용자 API, CLI, 기본값 또는 파일 포맷 변경을 문서에 반영했는가?
- 민감 정보나 불필요한 생성물이 변경에 포함되지 않았는가?


## Multi-agent 협업 절차

구현 범위가 넓거나 여러 독립 검증이 필요한 작업은 `planner → actor → evaluator → root` 순서로 진행한다. 역할은 병렬 구현 경쟁이 아니라, 명확한 handoff와 품질 게이트를 위한 책임 분리다.

### 1. Planner

- 관련 README, architecture, requirements 및 현재 구현을 먼저 읽는다.
- `tasks/`에 역할별 계획 문서를 작성한다. 계획에는 범위, 제외 범위, 수용 기준, 테스트 항목, 문서 변경 필요 여부, 위험 요소를 포함한다.
- 문서 간 계약 충돌이나 사용자 결정이 필요한 사항은 구현 전에 root에 보고한다.
- planner는 구현 소스를 수정하지 않는다.

### 2. Actor

- planner가 확정한 범위만 구현한다. 범위가 바뀌면 root의 명시적 승인을 받는다.
- 테스트를 먼저 추가해 실패를 확인하고, 최소 구현 후 리팩터링한다.
- 공개 API, 파일 포맷, CLI 기본값·출력 변경은 관련 문서를 같은 변경에서 갱신한다.
- 작업 결과에 변경 파일, 테스트 명령과 결과, 남은 위험을 기록한다.

### 3. Evaluator

- actor 변경을 독립적으로 요구사항·설계·보안 원칙과 대조한다.
- 성공 경로뿐 아니라 오류, 경계값, 파일 손상, 동시성, CLI 종료 코드와 문서 계약을 점검한다.
- 발견 사항은 심각도와 재현/검증 방법을 포함해 `tasks/` 문서에 남긴다.
- evaluator는 원칙적으로 구현을 수정하지 않는다. 수정이 필요하면 root가 actor에게 후속 작업으로 할당한다.

### 4. Root 통합과 완료 기준

- evaluator의 차단 이슈를 해결한 뒤에만 작업을 완료 처리하거나 커밋·푸시한다.
- 각 회차가 끝나면 `tasks/PROGRESS.md`에 완료 범위, 검증 결과, 미해결 항목과 환경 제약을 갱신한다.
- 최소한 `gofmt`, `go test ./...`, `go vet ./...`, `git diff --check`를 실행한다. 변경 범위가 해당하면 `go test -race ./...`도 실행한다.
- sub-agent 권한 또는 환경 제약으로 직접 수정·실행하지 못한 항목은 root가 재현하거나, 검증하지 못한 이유와 영향을 명시한다.
