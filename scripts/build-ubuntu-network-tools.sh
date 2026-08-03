#!/usr/bin/env bash
set -euo pipefail

root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
tag=${1:-netlab/ubuntu-network-tools:24.04}

docker build --network host -t "$tag" "$root/templates/docker/ubuntu-network-tools"
printf '%s\n' "$tag"
