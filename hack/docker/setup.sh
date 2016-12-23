#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

LIB_ROOT=$(dirname "${BASH_SOURCE}")/..
source "$LIB_ROOT/libbuild/common/lib.sh"
source "$LIB_ROOT/libbuild/common/public_image.sh"

GOPATH=$(go env GOPATH)
SRC=$GOPATH/src
BIN=$GOPATH/bin
ROOT=$GOPATH

APPSCODE_ENV=${APPSCODE_ENV:-dev}
IMG=tillerc

DIST=$GOPATH/src/github.com/appscode/tillerc/dist
mkdir -p $DIST
if [ -f "$DIST/.tag" ]; then
	export $(cat $DIST/.tag | xargs)
fi

clean() {
    pushd $GOPATH/src/github.com/appscode/tillerc/hack/docker
	rm -rf tillerc
	popd
}

build_binary() {
	pushd $GOPATH/src/github.com/appscode/tillerc
	./hack/builddeps.sh
    ./hack/make.py build tillerc
	detect_tag $DIST/.tag
	popd
}

build_docker() {
	pushd $GOPATH/src/github.com/appscode/tillerc/hack/docker
	cp $DIST/tillerc/tillerc-linux-amd64 tillerc
	chmod 755 tillerc

	cat >Dockerfile <<EOL
FROM appscode/base:8.6

RUN set -x \
  && apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/* /usr/share/doc /usr/share/man /tmp/*

COPY tillerc /tillerc
ENTRYPOINT ["/tillerc"]
EOL
	local cmd="docker build -t appscode/$IMG:$TAG ."
	echo $cmd; $cmd

	rm tillerc Dockerfile
	popd
}

build() {
	build_binary
	build_docker
}

docker_push() {
	if [ "$APPSCODE_ENV" = "prod" ]; then
		echo "Nothing to do in prod env. Are you trying to 'release' binaries to prod?"
		exit 0
	fi

    if [[ "$(docker images -q appscode/$IMG:$TAG 2> /dev/null)" != "" ]]; then
        docker push appscode/$IMG:$TAG
    fi
}

docker_release() {
	if [ "$APPSCODE_ENV" != "prod" ]; then
		echo "'release' only works in PROD env."
		exit 1
	fi
	if [ "$TAG_STRATEGY" != "git_tag" ]; then
		echo "'apply_tag' to release binaries and/or docker images."
		exit 1
	fi

    if [[ "$(docker images -q appscode/$IMG:$TAG 2> /dev/null)" != "" ]]; then
        docker push appscode/$IMG:$TAG
    fi
}

source_repo $@
