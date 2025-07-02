#!/usr/bin/env bash

# Copyright 2025 MongoDB Inc
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

PURLS_DEP_PATH="${PURLS_DEP_PATH:-build/package}"
PLUGIN_BINARY_PATH="${PLUGIN_BINARY_PATH:-./bin/plugin}"
PURLS_FILE="${PURLS_FILE:-purls.txt}"

echo "==> Generating dependency list"

platforms=(
  "linux arm64"
  "darwin arm64"
  "windows amd64"
)

for combo in "${platforms[@]}"; do
  read -r OS ARCH <<< "$combo"
  EXT=""
  [ "$OS" = "windows" ] && EXT=".exe"

  OUTFILE="${PURLS_DEP_PATH}/purls-${OS}_${ARCH}.txt"
  BINARY="${PLUGIN_BINARY_PATH}${EXT}"

  echo "==> Writing dependencies to $OUTFILE"
  GOOS=$OS GOARCH=$ARCH go build -trimpath -mod=readonly -o "$BINARY" ./cmd/plugin > /dev/null 2>&1
  go version -m "$BINARY" | \
    awk '$1 == "dep" || $1 == "=>" { print "pkg:golang/" $2 "@" $3 }' | \
    LC_ALL=C sort > "$OUTFILE"
done

MERGED="${PURLS_DEP_PATH}/${PURLS_FILE}"
echo "==> Merging dependencies to $MERGED"
cat "${PURLS_DEP_PATH}"/purls-*.txt | LC_ALL=C sort | uniq > "$MERGED"

echo "==> Cleaning up"
rm -f "${PURLS_DEP_PATH}"/purls-*.txt
