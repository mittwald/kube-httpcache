#!/bin/bash

set -e
VERSION=$1

if [ "${VERSION}" == "" ]; then
    echo "usage: $0 <version>"
fi

echo "Building kube-httpcache ${VERSION}"

# clean check
STATUS=$(git status --porcelain)
if [ "${STATUS}" != "" ]; then
    echo ${STATUS}
    echo "Repository is not clean"
    exit 1
fi

# git tagging
git tag ${VERSION}
git push --tags

# docker build
docker build -t re-docker-registry.ihrprod.net/kube-httpcache:${VERSION} -f build/package/docker/Dockerfile .
docker push re-docker-registry.ihrprod.net/kube-httpcache:${VERSION}
