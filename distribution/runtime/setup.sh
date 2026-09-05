#!/usr/bin/env bash
# Build the preview from this checkout. No host Go or administrator access.
set -euo pipefail
root=$(cd "$(dirname "$0")/../.." && pwd -P)
case "$(uname -s)/$(uname -m)" in
  Darwin/arm64) target=darwin-arm64 ;;
  Darwin/x86_64) target=darwin-amd64 ;;
  Linux/x86_64) target=linux-amd64 ;;
  *) echo 'Supported: macOS Intel/Apple Silicon and Linux x64.' >&2; exit 1 ;;
esac
if ! command -v docker >/dev/null 2>&1; then
  for directory in "$HOME/.docker/bin" /usr/local/bin /Applications/Docker.app/Contents/Resources/bin; do
    if [ -x "$directory/docker" ]; then export PATH="$directory:$PATH"; break; fi
  done
fi
if ! command -v docker >/dev/null 2>&1; then
  echo 'Install and start Docker Desktop (Mac) or Docker Engine (Linux), then retry.' >&2
  exit 1
fi
if [ "$(docker info --format '{{.OSType}}')" != linux ]; then
  echo 'Start a local Docker engine using Linux containers.' >&2; exit 1
fi
if ! git -C "$root" diff --quiet HEAD --; then
  echo 'Commit or stash tracked source changes before building an identifiable runtime.' >&2; exit 1
fi
revision=$(git -C "$root" rev-parse --short=12 HEAD)
image="gobble-runtime:$revision"
build_dir=$(mktemp -d)
trap 'rm -rf "$build_dir"' EXIT
docker build --platform linux/amd64 -f "$root/distribution/runtime/Dockerfile" --target runtime -t "$image" "$root"
docker build --platform linux/amd64 -f "$root/distribution/runtime/Dockerfile" --target launcher-artifacts --output "$build_dir" "$root"
# This also diagnoses missing amd64 emulation before installing the launcher.
docker run --rm --platform linux/amd64 "$image" version
install_dir="$HOME/.local/bin"
config_dir="$HOME/.local/share/gobble"
mkdir -p "$install_dir" "$config_dir"
install -m 755 "$build_dir/$target/gobble" "$install_dir/gobble"
printf 'export PATH=%q:$PATH\nexport GOBBLE_RUNTIME_IMAGE=%q\n' "$install_dir" "$image" > "$config_dir/env.sh"
printf '\nGobble installed. Run in this terminal (and each new terminal):\n\n  source %q\n\nThen prepare a real assay:\n\n  gobble demo rnaseq my-rnaseq\n  cd my-rnaseq\n  gobble doctor\n' "$config_dir/env.sh"
