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

: "${AWS_ACCESS_KEY_ID:?Missing AWS_ACCESS_KEY_ID}"
: "${AWS_SECRET_ACCESS_KEY:?Missing AWS_SECRET_ACCESS_KEY}"
: "${AWS_SESSION_TOKEN:?Missing AWS_SESSION_TOKEN}"

AWS_ACCOUNT_ID="${AWS_ACCOUNT_ID:-901841024863}"
AWS_REGION="${AWS_DEFAULT_REGION:-us-east-1}"

SILKBOMB_REGISTRY="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_REGION}.amazonaws.com"
SILKBOMB_REPO="release-infrastructure/silkbomb"
SILKBOMB_TAG="2.0"
SILKBOMB_IMAGE="${SILKBOMB_REGISTRY}/${SILKBOMB_REPO}:${SILKBOMB_TAG}"
SILKBOMB_PURLS="/pwd/purls.txt"
SILKBOMB_SBOM_OUT="/pwd/sbom.json"

if ! podman image exists "${SILKBOMB_IMAGE}"; then
  echo "Logging in to ECR..."
  aws ecr get-login-password --region "${AWS_REGION}" | \
    podman login --username AWS --password-stdin "${SILKBOMB_REGISTRY}"
fi

echo "Generating SBOMs"
podman run --rm \
  --pull=missing \
  -v "$(pwd):/pwd" \
  "${SILKBOMB_IMAGE}" \
  update \
  --purls "${SILKBOMB_PURLS}" \
  --sbom-out "${SILKBOMB_SBOM_OUT}"

echo "SBOM generated at $(pwd)/sbom.json"
