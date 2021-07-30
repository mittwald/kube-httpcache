# build kube-httpcache go app
FROM        golang:1.15.2 AS app-builder

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

WORKDIR     /workspace

COPY        . .

RUN         CGO_ENABLED=0 GOOS=linux \
            go build \
                -mod=vendor \
                -o kube-httpcache \
                ./cmd/kube-httpcache/main.go

# final image
FROM        ubuntu:18.04 AS final

WORKDIR     /
RUN         apt-get update && apt-get install -y gcc libedit2 libpcre++ ca-certificates && rm -rf /var/lib/apt/lists/*
COPY        --from=re-docker-registry.ihrprod.net/varnish:6.5 /opt/varnish /opt/varnish/
COPY        --from=app-builder /workspace/kube-httpcache .
#COPY        ./prometheus_varnish_exporter-1.6.linux-amd64/prometheus_varnish_exporter /usr/bin
ENV         PATH=/opt/varnish/bin:$PATH

ENTRYPOINT  [ "/kube-httpcache" ]
