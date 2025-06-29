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

release_date=${DATE:-$(date -u '+%Y-%m-%d')}
export DATE="${release_date}"

if [ -z "${AUTHOR:-}" ]; then
  AUTHOR=$(git config user.name)
fi

if [ -z "${VERSION:-}" ]; then
  VERSION=$(git tag --list 'atlascli/v*' --sort=-taggerdate | head -1 | cut -d 'v' -f 2)
fi

export AUTHOR VERSION
REPORT_OUT="${REPORT_OUT:-ssdlc-compliance-report-${VERSION}-${DATE}}"
echo "Generating SSDLC checklist for atlas-cli-plugin version ${VERSION}, author ${AUTHOR} and release date ${DATE}..."
echo "Report will be stored at ${REPORT_OUT}"
envsubst < docs/releases/ssdlc-compliance.template.md > ${REPORT_OUT}
