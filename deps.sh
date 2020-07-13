# when specifying a commit that is not on the main repository yet but in a PR,
# you'll have to specify the GitHUB PR number so that we can fetch it 

export RPMS_COMMIT=maistra-1.2
export RPMS_PR=
export IMAGES_COMMIT=maistra-1.2
export IMAGES_PR=

export ISTIO_COMMIT=maistra-1.2
export ISTIO_PR=
export ISTIO_CNI_COMMIT=maistra-1.2
export ISTIO_CNI_PR=
export IOR_COMMIT=maistra-1.2
export IOR_PR=

# this one's ignored for now as we're not currently rebuilding istio-proxy
# due to its 5h build time
export ISTIO_PROXY_COMMIT=maistra-1.2
# instead, we're using a fixed build of istio-proxy
export PROXY_VERSION=1.1.1

export PROMETHEUS_COMMIT=maistra-1.1
export PROMETHEUS_PR=

# we leave this blank to use whatever's defined in the .spec
export GRAFANA_VERSION=
