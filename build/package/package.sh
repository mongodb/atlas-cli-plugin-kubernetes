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

set -Eeou pipefail

export GOROOT="${GOROOT:?}"
export GORELEASER_KEY=${goreleaser_key:?}
export VERSION_GIT=${version:?}
VERSION=$(git tag --list 'v*' --sort=-taggerdate | head -1 | cut -d 'v' -f 2)
export VERSION
export GITHUB_REPOSITORY_OWNER="mongodb"
export GITHUB_REPOSITORY_NAME="atlas-cli-plugin-kubernetes"

make generate-all-manifests

# avoid race conditions on the notarization step by using `-p 1`
./bin/goreleaser release --config "build/package/.goreleaser.yml" --clean -p 1