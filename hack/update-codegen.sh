#!/usr/bin/env bash

# Apache-2.0, Copyright 2023 The Kubernetes Authors
# Modified for CRD generation and controller installation

set -o errexit
set -o nounset
set -o pipefail

GO_CMD=${1:-go}
PKG_ROOT=$(realpath "$(dirname ${BASH_SOURCE[0]})/..")
CODEGEN_PKG=$($GO_CMD list -m -f "{{.Dir}}" k8s.io/code-generator)

cd $PKG_ROOT

# Install controller-gen if not installed
echo "Installing controller-gen..."
go install sigs.k8s.io/controller-tools/cmd/controller-gen@latest

# First, generate helpers and client code as before using the code-generator
source "${CODEGEN_PKG}/kube_codegen.sh"

# Generate the client code (if needed)
kube::codegen::gen_helpers \
  --boilerplate /dev/null \
  "${PKG_ROOT}/pkg/apis"

kube::codegen::gen_client \
  --output-dir "${PKG_ROOT}/pkg/generated" \
  --output-pkg "github.com/frsarker/crd/pkg/generated" \
  --boilerplate /dev/null \
  --with-watch \
  --with-applyconfig \
  "${PKG_ROOT}/pkg/apis"

# Clean up temporary libraries added in go.mod by code-generator
"${GO_CMD}" mod tidy

# Automatically generate the CRD manifest from your Go types
echo "Generating CRD manifest from Go types..."
controller-gen crd:crdVersions=v1 paths=./pkg/apis/frsarker.dev/v1 output:crd:dir=./
# Optionally, expose the controller via a service (if needed)
# kubectl expose deployment <controller-deployment-name> --type=LoadBalancer --name=<service-name>

echo "Controller installation complete!"
