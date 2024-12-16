FROM        golang:1.20-bullseye AS builder

WORKDIR     /workspace
COPY        . .
RUN         CGO_ENABLED=0 GOOS=linux \
            go build \
                -installsuffix cgo \
                -o kube-httpcache \
                -a cmd/kube-httpcache/main.go

FROM        golang:1.20-bullseye AS builder-exporter
ENV         EXPORTER_VERSION=1.6.1

WORKDIR     /workspace
RUN         apt-get update \
            && apt-get install -y \
            git
RUN         git clone https://github.com/jonnenauha/prometheus_varnish_exporter.git --depth 1 --branch ${EXPORTER_VERSION}
WORKDIR     /workspace/prometheus_varnish_exporter
RUN         go build


FROM        debian:bullseye-slim AS final

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

WORKDIR     /

RUN         apt-get -qq update && apt-get -qq upgrade && apt-get -qq install curl && \
            curl -s https://packagecloud.io/install/repositories/varnishcache/varnish76/script.deb.sh | bash && \
            apt-get -qq update && apt-get -qq install varnish && \
            apt-get -qq purge curl gnupg && \
            apt-get -qq autoremove && apt-get -qq autoclean && \
            rm -rf /var/cache/*

RUN         mkdir /exporter && chown varnish /exporter

COPY        --from=builder /workspace/kube-httpcache .
COPY        --from=builder-exporter /workspace/prometheus_varnish_exporter/prometheus_varnish_exporter /exporter/

ENTRYPOINT  [ "/kube-httpcache" ]
