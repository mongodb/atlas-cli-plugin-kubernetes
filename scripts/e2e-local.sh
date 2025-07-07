#!/bin/bash


set -euo pipefail

if [[ -z "${MCLI_OPS_MANAGER_URL:-}" ]]; then
    echo "MCLI_OPS_MANAGER_URL is not set. Please set it to the desired version."
    exit 1
fi

if [[ -z "${MCLI_ORG_ID:-}" ]]; then
    echo "MCLI_ORG_ID is not set. Please set it to the desired version."
    exit 1
fi

if [[ -z "${MCLI_PUBLIC_KEY:-}" ]]; then
    echo "MCLI_PUBLIC_KEY is not set. Please set it to the desired version."
    exit 1
fi

if [[ -z "${MCLI_PRIVATE_KEY:-}" ]]; then
    echo "MCLI_PRIVATE_KEY is not set. Please set it to the desired version."
    exit 1
fi

export E2E_PLUGIN_BINARY_PATH=../bin/atlas-cli-plugin-kubernetes
export E2E_ATLASCLI_BINARY_PATH=../test/bin/atlascli

echo "==> Running E2E tests with Atlas CLI plugin for Kubernetes..."

