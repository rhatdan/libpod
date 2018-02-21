#!/bin/bash
set -xeuo pipefail

DIST=${DIST:=Fedora}
CONTAINER_RUNTIME=${CONTAINER_RUNTIME:=docker}
IMAGE=fedorapodmanbuild
PYTHON=python3
if [[ ${DIST} != "Fedora" ]]; then
    IMAGE=centospodmanbuild
    PYTHON=python
fi

# Build the test image
${CONTAINER_RUNTIME} build -t ${IMAGE} -f Dockerfile.${DIST} .

# Run the tests
${CONTAINER_RUNTIME} run --rm --privileged --net=host -v $PWD:/go/src/github.com/projectatomic/libpod --workdir /go/src/github.com/projectatomic/libpod -e PYTHON=$PYTHON -e STORAGE_OPTIONS="--storage-driver=vfs" -e CRIO_ROOT="/go/src/github.com/projectatomic/libpod" -e PODMAN_BINARY="/usr/bin/podman" -e CONMON_BINARY="/usr/libexec/crio/conmon" -e DIST=$DIST $IMAGE sh .papr.sh
