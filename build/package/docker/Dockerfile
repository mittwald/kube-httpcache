FROM        golang:1.14 AS builder

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

WORKDIR     /workspace

COPY        . .

RUN         CGO_ENABLED=0 GOOS=linux \
            go build \
                -installsuffix cgo \
                -o kube-httpcache \
                -a cmd/kube-httpcache/main.go


FROM        ubuntu:18.04 AS final

WORKDIR     /

RUN         apt-get update && \
            apt-get install -y varnish varnish-modules net-tools curl && \
            rm -rf /var/lib/apt/lists/*

COPY        ./prometheus_varnish_exporter-1.5.2.linux-amd64/prometheus_varnish_exporter /usr/bin
COPY        --from=builder /workspace/kube-httpcache .

ENTRYPOINT  [ "/kube-httpcache" ]
