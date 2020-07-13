set -ex

if [ -n "${PULL_PULL_SHA}" ]; then
    export ISTIO_OPERATOR_COMMIT=${PULL_PULL_SHA}
else
    export ISTIO_OPERATOR_COMMIT=$(git rev-parse HEAD)
fi

if [ -n "${PULL_NUMBER}" ]; then
    export ISTIO_OPERATOR_PR=${PULL_NUMBER}
fi

export HUB=${HUB:-quay.io/maistra-dev}
export REPO=https://copr.fedorainfracloud.org/coprs/g/maistra-dev/istio/repo/epel-8/group_maistra-dev-istio-epel-8.repo
export VERSION=1.2.${ISTIO_OPERATOR_COMMIT}

WORKDIR=$(mktemp -d)
echo "Created temp directory ${WORKDIR}"

function cleanup() {
    echo "Cleaning up..."
    popd
    echo "Removing temp directory ${WORKDIR}"
    rm -rf ${WORKDIR}
}

pushd ${WORKDIR}

trap cleanup EXIT

# build RPMs

git clone https://github.com/maistra/rpms.git
pushd rpms
if [ -n "${RPMS_PR}" ]; then
    git fetch origin pull/${RPMS_PR}/head:pull/${RPMS_PR}
    RPMS_COMMIT=${RPMS_COMMIT:-pull/${RPMS_PR}}
fi
git checkout ${RPMS_COMMIT}
make update
DEV_VERSION=${VERSION} make build-copr
popd

# build containers

git clone https://github.com/maistra/istio-images-centos.git
pushd istio-images-centos
if [ -n "${IMAGES_PR}" ]; then
    git fetch origin pull/${IMAGES_PR}/head:pull/${IMAGES_PR}
    IMAGES_COMMIT=${IMAGES_COMMIT:-pull/${IMAGES_PR}}
fi
git checkout ${IMAGES_COMMIT}
./create-images.sh -h ${HUB} -t ${VERSION} -bp
popd

## when done, comment hub/tag on the PR
GITHUB_TOKEN=$(cat /creds-github/github-token 2> /dev/null)
if [[ -n "${GITHUB_TOKEN}" && -n "${PULL_NUMBER}" ]]; then
    curl -H "Content-Type:application/json" -H "Authorization:token ${GITHUB_TOKEN}" \
        --data '{"body":"Release build complete. Find images at:\n\n```yaml\nhub: '"${HUB}"'\ntag: '"${VERSION}"'\n```\n"}' \
        https://api.github.com/repos/maistra/istio-operator/issues/${PULL_NUMBER}/comments
fi
