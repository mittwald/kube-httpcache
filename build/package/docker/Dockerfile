FROM        golang:1.14 AS builder

WORKDIR     /workspace

COPY        . .

RUN         CGO_ENABLED=0 GOOS=linux \
            go build \
                -installsuffix cgo \
                -o kube-httpcache \
                -a cmd/kube-httpcache/main.go


FROM        centos:7 AS final

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

WORKDIR     /

RUN         yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm && \
            yum install -y pygpgme yum-utils

COPY        assets/varnish.repo /etc/yum.repos.d/varnishcache_varnish60.repo

RUN         yum -q makecache -y --disablerepo='*' --enablerepo='varnishcache_varnish60' && \
            yum install -y varnish

COPY        --from=builder /workspace/kube-httpcache .

ENTRYPOINT  [ "/kube-httpcache" ]