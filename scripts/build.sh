#!/usr/bin/env bash
set -euo pipefail

go_executable="go"
output_directory="dist"

usage() {
    printf 'Usage: %s [--go PATH] [--output DIRECTORY]\n' "$0"
}

while (($# > 0)); do
    case "$1" in
        --go)
            [[ $# -ge 2 ]] || { usage >&2; exit 2; }
            go_executable="$2"
            shift 2
            ;;
        --output)
            [[ $# -ge 2 ]] || { usage >&2; exit 2; }
            output_directory="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *)
            printf 'Unknown argument: %s\n' "$1" >&2
            usage >&2
            exit 2
            ;;
    esac
done

source_directory="$(pwd -P)"
git_commit="$(git -C "$source_directory" rev-parse HEAD)" || {
    printf 'Unable to determine the git commit for the build source.\n' >&2
    exit 1
}
if [[ -n "$(git -C "$source_directory" status --porcelain --untracked-files=normal)" ]]; then
    git_dirty="true"
else
    git_dirty="false"
fi

mkdir -p "$output_directory"
output_directory="$(cd "$output_directory" && pwd -P)"

CGO_ENABLED=0 GOOS=windows GOARCH=amd64 "$go_executable" build -trimpath -o "$output_directory/conductor.exe" ./cmd/conductor
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 "$go_executable" build -trimpath -o "$output_directory/conductor-linux-amd64" ./cmd/conductor

(
    cd "$output_directory"
    sha256sum conductor.exe conductor-linux-amd64 > SHA256SUMS
)

{
    printf 'git_commit=%s\n' "$git_commit"
    printf 'git_dirty=%s\n' "$git_dirty"
    printf 'generated_utc=%s\n' "$(date -u +'%Y-%m-%dT%H:%M:%SZ')"
    printf 'targets=windows/amd64,linux/amd64\n'
    printf 'cgo_enabled=0\n'
} > "$output_directory/build-metadata.txt"
