#!/usr/bin/env bash

set -o errexit
set -o pipefail

if ! which go > /dev/null; then
	echo "golang needs to be installed"
	exit 1
fi

: ${SOURCE_DIR:=$(pwd)}
: ${OUT_DIR:=${SOURCE_DIR}/tmp/_output}
BIN_DIR="${OUT_DIR}/bin"
mkdir -p ${BIN_DIR}
PROJECT_NAME="istio-operator"
REPO_PATH="github.com/maistra/istio-operator"
BUILD_PATH="${REPO_PATH}/cmd/manager"

: ${VERSION:=development}
LD_EXTRAFLAGS="-X ${REPO_PATH}/pkg/version.buildVersion=${VERSION}"

: ${GITREVISION:=$(git rev-parse --verify HEAD 2> /dev/null || echo "unknown")}
LD_EXTRAFLAGS+=" -X ${REPO_PATH}/pkg/version.buildGitRevision=${GITREVISION}"

if [ -z "${GITSTATUS}" ]; then
  GITSTATUS="$(git diff-index --quiet HEAD -- 2> /dev/null; echo $?)"
  if [ "${GITSTATUS}" == "0" ]; then
    GITSTATUS="Clean"
  elif [ "${GITSTATUS}" == "1" ]; then
    GITSTATUS="Modified"
  else
    GITSTATUS="unknown"
  fi
fi
LD_EXTRAFLAGS+=" -X ${REPO_PATH}/pkg/version.buildStatus=${GITSTATUS}"

: ${GITTAG:=$(git describe 2> /dev/null || echo "unknown")}
LD_EXTRAFLAGS+=" -X ${REPO_PATH}/pkg/version.buildTag=${GITTAG} -X ${REPO_PATH}/pkg/version.minimumSupportedVersion=${MINIMUM_SUPPORTED_VERSION}"

LDFLAGS="-extldflags -static ${LD_EXTRAFLAGS}"

echo "building "${PROJECT_NAME}"..."
GO111MODULE=on GOOS=linux CGO_ENABLED=0 go build -mod=vendor -o "${BIN_DIR}/${PROJECT_NAME}" -ldflags "${LDFLAGS}" "$BUILD_PATH"
