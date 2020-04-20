FROM        golang:1.14 AS builder

WORKDIR     /workspace

COPY        . .

RUN         CGO_ENABLED=0 GOOS=linux \
            go build \
                -installsuffix cgo \
                -o kube-httpcache \
                -a cmd/kube-httpcache/main.go


FROM        centos:8 AS final

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

WORKDIR     /

RUN         yum install -y varnish varnish-modules net-tools

COPY        --from=builder /workspace/kube-httpcache .

ENTRYPOINT  [ "/kube-httpcache" ]