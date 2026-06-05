# GitHub Setup

This project is configured for the GitHub repository:

```text
github.com/mohamedhabibwork/abp-cli
```

Keep the `module` line in `go.mod` aligned with the final GitHub owner and repository name. If the repository moves, update it before publishing:

```bash
go mod edit -module github.com/<owner>/<repo>
go mod tidy
```

## Client Install Options

Install from GitHub with the setup script:

```bash
curl -fsSL https://raw.githubusercontent.com/mohamedhabibwork/abp-cli/main/install.sh | sh
```

Install from a source checkout:

```bash
sh ./install.sh
```

Install into a custom bin directory:

```bash
INSTALL_DIR="$HOME/.local/bin" sh ./install.sh
```

Install a tagged GitHub version:

```bash
VERSION=v0.1.0 sh ./install.sh
```

Install directly with Go:

```bash
go install github.com/mohamedhabibwork/abp-cli@latest
```

Windows PowerShell:

```powershell
.\install.ps1
.\install.ps1 -InstallDir "$HOME\.local\bin"
.\install.ps1 -Version v0.1.0
```

## GitHub Actions

The repository includes:

```text
.github/workflows/ci.yml
.github/workflows/release.yml
```

CI runs on pushes and pull requests. It runs `go test ./...` and builds the CLI.

The release workflow runs when a tag like `v0.1.0` is pushed. It builds binaries for:

```text
darwin/amd64
darwin/arm64
linux/amd64
linux/arm64
windows/amd64
windows/arm64
```

No custom secrets are required. The workflow uses GitHub's built-in `GITHUB_TOKEN` with `contents: write` permission to create the release.

Every published version must pass the release quality gate first:

```bash
make release-check
```

The GitHub release workflow runs the same gate before creating auto tags, building archives, or publishing release assets.

Recommended repository rules:

```text
main/master branch
  Require pull request before merge.
  Require status checks before merge.
  Require pull-request / validate-title.
  Require pull-request / quality.
  Require ci / test.
  Block force pushes.

v* tags
  Allow release tags only for maintainers and GitHub Actions.
  Use the release workflow for published versions.
```

Direct tag pushes still run the release quality gate. If the gate fails, GitHub may already have the tag, but no GitHub Release or release assets are published.

Pull request and issue automation is also included:

```text
.github/workflows/pr.yml
.github/workflows/issue-management.yml
.github/PULL_REQUEST_TEMPLATE.md
.github/ISSUE_TEMPLATE/
.github/dependabot.yml
```

The PR workflow validates the title, formatting, module tidiness, vetting, tests, builds, and installer scripts. The issue workflow creates common labels when needed and applies triage labels automatically.

## Initial Repository Setup

Create the GitHub repository, then push this code:

```bash
git init
git add .
git commit -m "Initial ABP CLI"
git branch -M main
git remote add origin git@github.com:mohamedhabibwork/abp-cli.git
git push -u origin main
```

Use the HTTPS remote instead if your client does not use SSH keys:

```bash
git remote add origin https://github.com/mohamedhabibwork/abp-cli.git
```

## Publish A Release

Automatic release from `main` or `master`:

```bash
git push origin main
```

The workflow calculates the next semantic version from Conventional Commits, creates the tag, builds archives, generates release notes, and publishes the GitHub release.

It also updates `CHANGELOG.md` for every version. For automatic and manual branch releases, the workflow commits the changelog first and then tags that changelog commit. For direct `v*` tag pushes, the workflow cannot rewrite the existing tag, so it generates release notes and uploads `CHANGELOG.md` as a release asset.

Explicit release tag from a clean checkout:

```bash
go test ./...
mkdir -p dist
go build -trimpath -ldflags="-s -w" -o dist/abp-cli .
git tag v0.1.0
git push origin v0.1.0
```

After the tag is pushed, GitHub Actions creates the release and uploads binaries plus `checksums.txt`.

Release rules are documented in [.github/release-rules.md](../.github/release-rules.md).

Clients can then install by Go module:

```bash
go install github.com/mohamedhabibwork/abp-cli@v0.1.0
```

Or download the matching binary from the GitHub release page.
