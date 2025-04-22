###############
# Stage 1: Build kube-httpcache
###############
ARG GO_VERSION=1.24
FROM public.ecr.aws/docker/library/golang:${GO_VERSION}-bullseye AS builder
WORKDIR /workspace
COPY . .
RUN CGO_ENABLED=0 GOOS=linux \
    go build -installsuffix cgo -o kube-httpcache -a cmd/kube-httpcache/main.go
###############
# Stage 2: Build Prometheus Varnish Exporter
###############
ARG GO_VERSION=1.24
FROM public.ecr.aws/docker/library/golang:${GO_VERSION}-bullseye AS builder-exporter
ARG EXPORTER_VERSION=1.6.1
WORKDIR /workspace
RUN apt-get update && \
    apt-get install -y --no-install-recommends git ca-certificates && \
    git clone https://github.com/jonnenauha/prometheus_varnish_exporter.git --branch ${EXPORTER_VERSION} --depth 1 && \
    cd prometheus_varnish_exporter && \
    go build -o /workspace/prometheus_varnish_exporter && \
    apt-get purge -y git && \
    apt-get autoremove -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/*
###############
# Stage 3: Final Image with Varnish 7.7
###############
FROM public.ecr.aws/docker/library/debian:bullseye-slim AS final
ARG VARNISH_REPO_VERSION=77
WORKDIR /
RUN apt-get update && \
    apt-get install -y --no-install-recommends curl gnupg ca-certificates && \
    curl -fsSL https://packagecloud.io/install/repositories/varnishcache/varnish${VARNISH_REPO_VERSION}/script.deb.sh | bash && \
    apt-get update && \
    apt-get install -y --no-install-recommends varnish && \
    apt-get purge -y curl gnupg && \
    apt-get autoremove -y && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/* /tmp/* /var/tmp/* /var/cache/*
RUN mkdir -p /exporter && chown -R varnish /exporter
COPY --from=builder /workspace/kube-httpcache .
COPY --from=builder-exporter /workspace/prometheus_varnish_exporter /exporter/

ENTRYPOINT ["/kube-httpcache"]
