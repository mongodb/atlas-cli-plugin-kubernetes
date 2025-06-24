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

export AWS_DEFAULT_REGION=us-east-1

: "${AWS_ACCOUNT_ID:?Missing AWS_ACCOUNT_ID}"
: "${AWS_ACCESS_KEY_ID:?Missing AWS_ACCESS_KEY_ID}"
: "${AWS_SECRET_ACCESS_KEY:?Missing AWS_SECRET_ACCESS_KEY}"

export SILKBOMB_IMAGE="${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_DEFAULT_REGION}.amazonaws.com/release-infrastructure/silkbomb:2.0"

aws ecr get-login-password --region "${AWS_DEFAULT_REGION}" | \
  podman login --username AWS --password-stdin "${AWS_ACCOUNT_ID}.dkr.ecr.${AWS_DEFAULT_REGION}.amazonaws.com"

echo "Obtaining Kondukto credentials from secret manager in AWS"
kondukto_token=$(aws secretsmanager get-secret-value \
  --secret-id "kondukto-token" \
  --query 'SecretString' \
  --output text)
echo "KONDUKTO_TOKEN=$kondukto_token" > kondukto_credentials.env

echo "Generating SBOMs"
podman run --rm \
  -v "$(pwd):/pwd" \
  "${SILKBOMB_IMAGE}" \
  update \
  --purls /pwd/purls.txt \
  --sbom-out /pwd/sbom.json
echo "SBOM generated at $(pwd)/sbom.json"

echo "Uploading SBOM to Kondukto"
podman run --rm \
  -v "$(pwd):/pwd" \
  --env-file kondukto_credentials.env \
  "${SILKBOMB_IMAGE}" \
  upload \
  --sbom-in /pwd/sbom.json \
  --repo mongodb_atlas-cli-plugin-kubernetes \
  --branch main \
  --verbose
echo "Uploading complete"