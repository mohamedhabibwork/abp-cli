#!/usr/bin/env sh
set -eu

VERSION="${VERSION:-v0.0.0-check}"
COMMIT="${COMMIT:-release-check}"
BUILD_DATE="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

echo "Checking Go formatting..."
gofmt_files=$(find . -name '*.go' -not -path './dist/*' -not -path './abp-templates/*' -exec gofmt -l {} +)
if [ -n "$gofmt_files" ]; then
  echo "Run gofmt on these files:" >&2
  echo "$gofmt_files" >&2
  exit 1
fi

echo "Checking go.mod tidiness..."
go mod tidy -diff

echo "Running go vet..."
go vet ./...

echo "Running tests..."
go test ./...

echo "Checking installer scripts..."
sh -n install.sh
sh -n scripts/update-changelog.sh
if command -v pwsh >/dev/null 2>&1; then
  pwsh -NoProfile -Command '$errors = $null; $null = [System.Management.Automation.PSParser]::Tokenize((Get-Content -Raw ./install.ps1), [ref]$errors); if ($errors) { $errors | Format-List; exit 1 }'
else
  echo "PowerShell is not installed; skipping install.ps1 syntax check."
fi

echo "Building versioned smoke binary..."
tmp_dir=$(mktemp -d "${TMPDIR:-/tmp}/abp-cli-release-check.XXXXXX")
trap 'rm -rf "$tmp_dir"' EXIT INT TERM

module_path=$(go list -m)
ldflags="-s -w -X ${module_path}/internal/abpcrud.Version=${VERSION} -X ${module_path}/internal/abpcrud.Commit=${COMMIT} -X ${module_path}/internal/abpcrud.BuildDate=${BUILD_DATE}"
go build -trimpath -ldflags="$ldflags" -o "$tmp_dir/abp-cli" .

"$tmp_dir/abp-cli" version | grep -F "abp-cli ${VERSION}" >/dev/null
"$tmp_dir/abp-cli" templates --out "$tmp_dir/templates" >/dev/null

echo "Release gate passed for ${VERSION}."
