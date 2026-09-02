#!/usr/bin/env bash

# Copyright 2026 MongoDB Inc
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

# Logs the available container runtime into the DevProd Platforms ECR registry, which hosts the
# Garasign signing images and the Docker Hub pull-through cache. Needs AWS credentials in the
# environment; in Evergreen these come from ec2.assume_role.
# See https://docs.devprod.prod.corp.mongodb.com/devprod-platforms-ecr

set -Eeou pipefail

ECR_REGISTRY=${ECR_REGISTRY:-901841024863.dkr.ecr.us-east-1.amazonaws.com}
ECR_REGION=${ECR_REGION:-us-east-1}

password=$(aws ecr get-login-password --region "${ECR_REGION}")

# podman on RHEL CI, docker on local dev
for runtime in podman docker; do
    if command -v "${runtime}" > /dev/null 2>&1; then
        echo "${password}" | "${runtime}" login --username AWS --password-stdin "${ECR_REGISTRY}"
        exit 0
    fi
done

echo "no usable container runtime found" >&2
exit 1
