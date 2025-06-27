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
: "${KONDUKTO_CREDENTIALS_FILE:?Missing KONDUKTO_CREDENTIALS_FILE}"
: "${KONDUKTO_REPO:?Missing KONDUKTO_REPO}"
: "${KONDUKTO_BRANCH:?Missing KONDUKTO_BRANCH}"

# load the token
if [[ ! -s "${KONDUKTO_CREDENTIALS_FILE}" ]]; then
  echo "ERROR: credentials file is missing or empty: ${KONDUKTO_CREDENTIALS_FILE}" >&2
  exit 1
fi

if podman image exists "${SILKBOMB_IMAGE}"; then
  echo "Using existing local image: ${SILKBOMB_IMAGE}"
else # Else image will need to be pulled from AWS registry
  : "${AWS_ACCESS_KEY_ID:?Missing AWS_ACCESS_KEY_ID}"
  : "${AWS_SECRET_ACCESS_KEY:?Missing AWS_SECRET_ACCESS_KEY}"
  : "${AWS_SESSION_TOKEN:?Missing AWS_SESSION_TOKEN}"

  echo "Logging in to ECR..."
  aws ecr get-login-password --region us-east-1 | \
    podman login --username AWS --password-stdin "${SILKBOMB_REGISTRY}"
fi

echo "Uploading SBOM to Kondukto..."
podman run --rm \
  --pull=missing \
  -v "$(pwd):/pwd" \
  --env-file "${KONDUKTO_CREDENTIALS_FILE}" \
  "${SILKBOMB_IMAGE}" \
  upload \
  --sbom-in "${SILKBOMB_SBOM_FILE}" \
  --repo "${KONDUKTO_REPO}" \
  --branch "${KONDUKTO_BRANCH}"

echo "Uploading complete."
rm -f "${KONDUKTO_CREDENTIALS_FILE}"