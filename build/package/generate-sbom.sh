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

: "${SILKBOMB_IMAGE:?Missing SILKBOMB_IMAGE}"
: "${SILKBOMB_PURLS_FILE:?Missing SILKBOMB_PURLS_FILE}"
: "${SILKBOMB_SBOM_FILE:?Missing SILKBOMB_SBOM_FILE}"

# Resolve absolute path of the purls file directory
PURLS_DIR="$(cd "$(dirname "${SILKBOMB_PURLS_FILE}")" && pwd)"

# Check if image exists locally
if podman image exists "${SILKBOMB_IMAGE}"; then
  echo "Using existing local image: ${SILKBOMB_IMAGE}"
else # Else image will need to be pulled from AWS registry
  : "${AWS_ACCESS_KEY_ID:?Missing AWS_ACCESS_KEY_ID}"
  : "${AWS_SECRET_ACCESS_KEY:?Missing AWS_SECRET_ACCESS_KEY}"
  : "${AWS_SESSION_TOKEN:?Missing AWS_SESSION_TOKEN}"
  : "${SILKBOMB_REGISTRY:?Missing SILKBOMB_REGISTRY}"

  echo "Logging in to ECR..."
  aws ecr get-login-password --region us-east-1 | \
    podman login --username AWS --password-stdin "${SILKBOMB_REGISTRY}"
fi

echo "Generating SBOMs with image: ${SILKBOMB_IMAGE}"
podman run --rm \
  --pull=missing \
  -v "${PURLS_DIR}:/pwd" \
  "${SILKBOMB_IMAGE}" \
  update \
  --purls "/pwd/$(basename "${SILKBOMB_PURLS_FILE}")" \
  --sbom-out "/pwd/$(basename "${SILKBOMB_SBOM_FILE}")"

echo "SBOM generated at ${SILKBOMB_SBOM_FILE}"
