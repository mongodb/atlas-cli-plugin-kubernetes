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

OS=$(uname -s)
ARCH=$(uname -m)

echo "==> Fetching AtlasCLI binary..."
rm -rf ./test/bin
mkdir -p ./test/bin

if [ "$OS" = "Darwin" ]; then
    if [ "$ARCH" = "arm64" ]; then
        curl -L https://fastdl.mongodb.org/mongocli/mongodb-atlas-cli_1.35.0_macos_arm64.zip -o ./test/bin/mongodb-atlas-cli.zip
    elif [ "$ARCH" = "x86_64" ]; then
        curl -L https://fastdl.mongodb.org/mongocli/mongodb-atlas-cli_1.35.0_macos_x86_64.zip -o ./test/bin/mongodb-atlas-cli.zip
    fi
    unzip -q ./test/bin/mongodb-atlas-cli.zip -d ./test/bin/tmp
elif [ "$OS" = "Linux" ]; then
    if [ "$ARCH" = "x86_64" ]; then
        curl -L https://fastdl.mongodb.org/mongocli/mongodb-atlas-cli_1.35.0_linux_x86_64.tar.gz -o ./test/bin/mongodb-atlas-cli.tar.gz
    elif [ "$ARCH" = "aarch64" ]; then
        curl -L https://fastdl.mongodb.org/mongocli/mongodb-atlas-cli_1.35.0_linux_arm64.tar.gz -o ./test/bin/mongodb-atlas-cli.tar.gz
    fi
    mkdir -p ./test/bin/tmp
    tar --strip-components=1 -xf ./test/bin/mongodb-atlas-cli.tar.gz -C ./test/bin/tmp
elif [[ "$OS" =~ "MINGW" ]] || [[ "$OS" =~ "MSYS_NT" ]] || [[ "$OS" =~ "CYGWIN_NT" ]]; then
    curl -L https://fastdl.mongodb.org/mongocli/mongodb-atlas-cli_1.35.0_windows_x86_64.zip -o ./test/bin/mongodb-atlas-cli.zip
    unzip -q ./test/bin/mongodb-atlas-cli.zip -d ./test/bin/tmp
else
    echo "Unsupported OS or architecture"
    exit 1
fi

# Move binary to ./test/bin
mv ./test/bin/tmp/bin/atlas* ./test/bin/

# # Clean up
rm -rf ./test/bin/tmp
rm  ./test/bin/mongodb-atlas-cli*

chmod +x ./test/bin/atlas*
