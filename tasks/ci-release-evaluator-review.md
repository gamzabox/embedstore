# CI 및 릴리스 자동화 Evaluator 검토

검토일: 2026-08-02 (KST)

## 검토 범위

- `tasks/ci-release-planner-plan.md`
- `.github/workflows/ci.yml`
- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `README.md`, `docs/REQUIREMENTS.md`, `tasks/PROGRESS.md`

## 결과

차단 이슈 없음. 구성은 계획의 CI 품질 게이트, 태그 기반 release, 5개 플랫폼 archive, SHA-256 checksum 계약을 충족한다.

## 확인 사항

- CI는 모든 `push`/`pull_request`에서 실행하며, top-level `contents: read`와 CI checkout의 `persist-credentials: false`로 최소 권한을 사용한다.
- CI 순서는 `gofmt -l` 검사, `go test ./...`, `go vet ./...`, `git diff --check`, `dominikh/staticcheck-action@v1`, Linux `go test -race ./...`로 요구사항과 일치한다. `setup-go`는 `go.mod`를 읽어 Go 1.22를 사용하므로 staticcheck 2024.1.1과 호환되는 범위다.
- Release는 `v*` 태그 push만 수신하며 `contents: write`와 `GITHUB_TOKEN` 전달로 GitHub Release 작성 권한을 갖는다. checkout full history와 Go version file 설정도 적절하다.
- GoReleaser 설정은 `version: 2`, `linux/{amd64,arm64}`, `darwin/{amd64,arm64}`, `windows/amd64`만 남기도록 Windows arm64를 제외한다. Unix 기본 `tar.gz`, Windows `zip`, `LICENSE` 포함 및 SHA-256 `checksums.txt`가 모두 정의됐다. `format_overrides.formats`와 `archives.ids`는 현 GoReleaser v2 스키마에 맞는다.
- README와 요구사항은 CI trigger 및 release 산출물/플랫폼을 같은 내용으로 설명한다. `PROGRESS.md`도 로컬 환경에서 실행할 수 없는 YAML/GoReleaser 검증을 명시한다.

## 비차단 후속 권고

1. release checkout에도 `persist-credentials: false`를 설정하면 Git config에 토큰을 남기지 않고 GoReleaser에만 `GITHUB_TOKEN`을 전달할 수 있다. 현재도 GoReleaser action에 토큰을 전달하므로 기능 차단은 아니지만, 권한 노출 범위를 더 줄이는 강화 항목이다.
2. GoReleaser CLI와 YAML validator가 이 환경에 없어 `goreleaser check` 및 `release --snapshot --clean`은 실행하지 못했다. 첫 일반 push/PR CI와 실제가 아닌 테스트 태그를 사용하는 release workflow 실행으로 YAML·action·token/repository 설정을 확인해야 한다. 이는 `PROGRESS.md`에 이미 정확히 기록돼 있다.
3. major action tag 사용은 계획상 허용되나, 공급망 통제를 강화할 때 commit SHA pinning과 Dependabot을 후속으로 검토할 수 있다.

## 실행한 검토 명령

- `git diff --check` 통과
- 로컬 도구 확인: `goreleaser`, YAML validator 미설치. 따라서 GoReleaser/schema 실행 검증은 수행하지 못함.

