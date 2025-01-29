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

# mac_notarize generated binaries with Apple and replace the original binary with the notarized one
# This depends on binaries being generated in a goreleaser manner and gon being set up.
# goreleaser should already take care of calling this script as a hook.

if [[ -f "./dist/macos_darwin_amd64_v1/atlas-cli-plugin-kubernetes" && -f "./dist/macos_darwin_arm64/atlas-cli-plugin-kubernetes" && ! -f "./dist/atlas-cli-plugin-kubernetes_macos_signed.zip" ]]; then
	echo "notarizing macOs binaries"
	zip -r ./dist/atlas-cli-plugin-kubernetes_amd64_arm64_bin.zip ./dist/macos_darwin_amd64_v1/atlas-cli-plugin-kubernetes ./dist/macos_darwin_arm64/atlas-cli-plugin-kubernetes # The Notarization Service takes an archive as input
	./linux_amd64/macnotary \
		-f ./dist/atlas-cli-plugin-kubernetes_amd64_arm64_bin.zip \
		-m notarizeAndSign -u https://dev.macos-notary.build.10gen.cc/api \
		-b com.mongodb.atlas-cli-plugin-kubernetes \
		-o ./dist/atlas-cli-plugin-kubernetes_macos_signed.zip

	echo "replacing original files"
	unzip -oj ./dist/atlas-cli-plugin-kubernetes_macos_signed.zip dist/macos_darwin_amd64_v1/atlas-cli-plugin-kubernetes -d ./dist/macos_darwin_amd64_v1/
	unzip -oj ./dist/atlas-cli-plugin-kubernetes_macos_signed.zip dist/macos_darwin_arm64/atlas-cli-plugin-kubernetes -d ./dist/macos_darwin_arm64/
fi
