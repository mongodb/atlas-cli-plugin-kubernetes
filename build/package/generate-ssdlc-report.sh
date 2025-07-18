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

: "${TAG:=$(git tag --list 'v*' --sort=-taggerdate | head -1)}"
: "${VERSION:=${TAG#v}}"
: "${AUTHOR:=$(git for-each-ref --format '%(taggername)' "refs/tags/${TAG}")}"
: "${DATE:=$(git for-each-ref --format '%(taggerdate:short)' "refs/tags/${TAG}")}"

# Fallbacks in case tag is missing or doesn't have tagger metadata
AUTHOR="${AUTHOR:-Unknown Author}"
DATE="${DATE:-$(date -u '+%Y-%m-%d')}"
export AUTHOR VERSION DATE

REPORT_OUT="${REPORT_OUT:-ssdlc-compliance-report.md}"
echo "Generating SSDLC checklist for atlas-cli-plugin version ${VERSION}, author ${AUTHOR}, release date ${DATE}..."
echo "Report will be part of the release: ${REPORT_OUT}"

envsubst < docs/releases/ssdlc-compliance.template.md > "${REPORT_OUT}"
