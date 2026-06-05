#!/usr/bin/env sh
set -eu

CLI_NAME="${CLI_NAME:-abp-cli}"
MODULE_PATH="${MODULE_PATH:-github.com/mohamedhabibwork/abp-cli}"
VERSION="${VERSION:-}"
INSTALL_DIR="${INSTALL_DIR:-}"

script_dir=$(CDPATH= cd "$(dirname "$0")" && pwd)
local_module_path=""

if [ -f "${script_dir}/go.mod" ]; then
  local_module_path=$(awk '$1 == "module" { print $2; exit }' "${script_dir}/go.mod")
fi

if ! command -v go >/dev/null 2>&1; then
  echo "Go is required to install ${CLI_NAME}." >&2
  echo "Install Go, then run this script again: https://go.dev/doc/install" >&2
  exit 1
fi

if [ -n "$INSTALL_DIR" ]; then
  mkdir -p "$INSTALL_DIR"
  export GOBIN="$INSTALL_DIR"
fi

if [ -n "$VERSION" ]; then
  install_target="${MODULE_PATH}@${VERSION}"
  echo "Installing ${CLI_NAME} from ${install_target}..."
  go install "$install_target"
elif [ "$local_module_path" = "$MODULE_PATH" ] && [ -f "${script_dir}/main.go" ]; then
  echo "Installing ${CLI_NAME} from ${script_dir}..."
  (cd "$script_dir" && go install .)
else
  install_target="${MODULE_PATH}@latest"
  echo "Installing ${CLI_NAME} from ${install_target}..."
  go install "$install_target"
fi

bin_dir="${GOBIN:-$(go env GOPATH)/bin}"
bin_path="${bin_dir}/${CLI_NAME}"

if [ ! -x "$bin_path" ]; then
  echo "Install finished, but ${bin_path} was not found." >&2
  echo "Check GOBIN or GOPATH with: go env GOBIN GOPATH" >&2
  exit 1
fi

echo
echo "${CLI_NAME} installed:"
echo "  ${bin_path}"
echo

case ":${PATH}:" in
  *":${bin_dir}:"*) ;;
  *)
    echo "Add this directory to PATH:"
    echo "  export PATH=\"${bin_dir}:\$PATH\""
    echo
    ;;
esac

"$bin_path" --help >/dev/null
echo "Verified: ${CLI_NAME} --help"
