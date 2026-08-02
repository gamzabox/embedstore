# CI 및 릴리스 자동화 계획

## 범위

MVP 기능 변경 없이 GitHub Actions 기반의 품질 게이트와 태그 기반 배포 자동화를 추가한다.

- `.github/workflows/ci.yml`: push와 pull request에서 format, test, vet, staticcheck, Linux race test를 실행한다.
- `.github/workflows/release.yml`: `v*` 태그 push에서 GoReleaser를 실행해 GitHub Release와 플랫폼별 archive 및 `checksums.txt`를 만든다.
- `.goreleaser.yaml`: `cmd/embedstore`를 Linux amd64/arm64, macOS amd64/arm64, Windows amd64로 빌드하고 archive·checksum 산출물 이름을 정의한다.
- `tasks/PROGRESS.md`: 작업 시작·완료 상태와 검증, CI에서만 가능한 검증을 기록한다.
- `README.md`, `docs/REQUIREMENTS.md`: 사용자가 CI와 릴리스 사용 방법을 알아야 하는 최소 범위에서만 갱신한다. 요구사항의 이미 명시된 플랫폼/산출물 계약은 유지한다.

## 제외 범위

- 제품 Go 코드, 파일 포맷, CLI 동작 및 공개 API 변경
- interactive shell/evaluate/benchmark 기능
- npm, Docker, 서명·SBOM·Homebrew/Scoop 배포와 changelog 자동 생성
- 실제 release tag 생성 또는 GitHub Release 게시. 워크플로 정의만 추가한다.

## 확정할 설정

### CI

- Go 버전은 `go.mod`의 `go 1.22`를 기준으로 `1.22.x`를 사용한다. 버전을 중복 관리하지 않도록 후속적으로 `go-version-file: go.mod`를 우선한다.
- `actions/checkout@v4`, `actions/setup-go@v5`를 사용한다. checkout은 `persist-credentials: false`로 설정한다.
- workflow 권한은 top-level `permissions: contents: read`로 최소화한다.
- 트리거는 `push`와 `pull_request`이며, `dev/first`와 기본 브랜치 이름을 추측해 제한하지 않는다. 모든 브랜치/PR에서 동일하게 검증한다.
- Go 단계는 `gofmt -l` 결과가 비어 있는지 검사, `go test ./...`, `go vet ./...`, `go test -race ./...` 순서로 실행한다. Linux GitHub-hosted runner는 현 로컬 ThreadSanitizer VMA 제약을 우회하는 지원 환경이다.
- `git diff --check`를 별도 단계로 실행한다.
- staticcheck는 공식 action인 `dominikh/staticcheck-action@v1`로 실행하며 Go `1.22.x`를 명시한다. 개발 의존성이나 `go.mod` 변경 없이 도구 버전을 action이 격리한다.

### Release

- 태그 push `v*`에서만 실행한다. `workflow_dispatch`는 임의 버전 릴리스를 피하기 위해 이번 최소 범위에서는 추가하지 않는다.
- release workflow의 권한은 `contents: write`만 부여한다. `GITHUB_TOKEN`을 GoReleaser의 `GITHUB_TOKEN` 환경 변수로 전달한다.
- full history가 필요하므로 `actions/checkout@v4`에 `fetch-depth: 0`을 설정하고 `actions/setup-go@v5`는 `go-version-file: go.mod`를 사용한다.
- `goreleaser/goreleaser-action@v6`과 GoReleaser v2 stable을 사용한다. `args: release --clean`으로 이전 `dist` 잔여물을 제거한다.
- `.goreleaser.yaml`은 project name `embedstore`, main package `./cmd/embedstore`, binary `embedstore`를 정의한다. 대상은 `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/amd64`이다.
- archive format은 Unix 대상 `tar.gz`, Windows 대상 `zip`으로 한다. archive에는 binary와 `LICENSE`를 포함한다. `checksums.txt`는 SHA-256 기본값으로 모든 archive에 대해 생성한다.
- 현재 CLI는 version 출력이나 version 변수가 없으므로 `ldflags` 버전 주입은 추가하지 않는다. 이를 억지로 넣으면 제품 CLI/API 범위를 바꾸게 된다.
- repository에 `CHANGELOG.md`가 없으므로 GoReleaser changelog 생성은 비활성화한다. 릴리스 노트 자동 생성/관리 정책은 별도 사용자 결정이 필요하다.

## 수용 기준

1. CI workflow는 push/PR에서 format, `go test ./...`, `go vet ./...`, `git diff --check`, staticcheck, Linux race test를 실행한다.
2. CI는 최소 권한과 고정된 major action versions를 사용하며, 외부 credential을 요구하지 않는다.
3. release workflow는 `v*` 태그에서만 실행되고 `contents: write` 권한으로 GitHub Release를 게시할 수 있다.
4. GoReleaser 설정이 요구된 5개 플랫폼 조합을 빌드하고 OS별 archive 및 SHA-256 `checksums.txt`를 만든다.
5. 구성 파일은 로컬에서 YAML parser와 `goreleaser check`(설치 가능할 때), `goreleaser release --snapshot --clean`(게시 없이)로 검증한다. 설치 불가 시 CI가 최초 검증 지점임을 진행 문서에 남긴다.
6. 기존 `go test ./...`, `go vet ./...`, `git diff --check`가 통과한다. race는 로컬 환경에서 unsupported VMA로 실패하면 CI Linux runner에서의 실행으로 보완한다.

## 구현 순서와 테스트

1. actor는 우선 workflow/config의 형식과 대상 플랫폼을 검증하는 회귀 관점의 점검 항목을 정하고, `.github/workflows`와 `.goreleaser.yaml`을 작성한다.
2. YAML 문법을 검사하고 workflow의 event/permissions/job/command와 GoReleaser build/archive/checksum 정의를 검토한다.
3. GoReleaser CLI가 있으면 `goreleaser check`와 snapshot build를 실행한다. 없으면 임의 설치를 하지 않고, 실행하지 못한 사유를 기록한다.
4. Go 품질 명령을 다시 실행하고 `tasks/PROGRESS.md`에 완료 범위와 검증을 갱신한다.

## 위험과 결정

- GitHub Actions의 실제 실행, GitHub Release 작성, `GITHUB_TOKEN` 권한은 로컬에서 검증할 수 없다. 태그를 만들기 전 일반 push/PR CI와 test tag의 release workflow 결과를 확인해야 한다.
- tag protection 또는 repository Actions 권한이 `GITHUB_TOKEN`의 write를 제한하면 release job이 실패한다. 저장소 Settings의 Actions workflow permissions가 release 작성 권한을 허용해야 한다.
- 액션은 major tag가 시간이 지나며 변경될 수 있다. 공급망을 더 엄격히 관리하려면 후속 작업에서 commit SHA pinning과 Dependabot을 도입한다.
- GoReleaser v2 schema는 설치 버전과 일치해야 한다. actor는 action의 안정 버전과 `.goreleaser.yaml` schema를 맞추고 snapshot check로 확인한다.
