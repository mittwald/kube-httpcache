# ------------------------
# Stage 1: Build kube-httpcache
# ------------------------
ARG GO_VERSION=1.24
FROM public.ecr.aws/docker/library/golang:${GO_VERSION}-bullseye AS builder

WORKDIR /workspace
COPY . .

RUN CGO_ENABLED=0 GOOS=linux \
    go build -installsuffix cgo -o kube-httpcache -a cmd/kube-httpcache/main.go

# ------------------------
# Stage 2: Build Prometheus Varnish Exporter
# ------------------------
ARG GO_VERSION=1.24
FROM public.ecr.aws/docker/library/golang:${GO_VERSION}-bullseye AS builder-exporter
ARG EXPORTER_VERSION=1.6.1
WORKDIR /workspace
RUN apt-get update && apt-get install -y --no-install-recommends git && \
    git clone https://github.com/jonnenauha/prometheus_varnish_exporter.git --branch ${EXPORTER_VERSION} --depth 1

WORKDIR /workspace/prometheus_varnish_exporter
RUN go build

# ------------------------
# Stage 3: Final image with Varnish 7.7
# ------------------------
ARG VARNISH_REPO_VERSION=77
FROM public.ecr.aws/docker/library/debian:bullseye-slim AS final

WORKDIR /

# Install Varnish from packagecloud (e.g., varnish77)
RUN apt-get update && apt-get install -y --no-install-recommends curl gnupg ca-certificates && \
    curl -s https://packagecloud.io/install/repositories/varnishcache/varnish${VARNISH_REPO_VERSION}/script.deb.sh | bash && \
    apt-get update && apt-get install -y --no-install-recommends varnish && \
    apt-get -y purge curl gnupg && apt-get autoremove -y && apt-get clean && rm -rf /var/lib/apt/lists/* /var/cache/*

# Create directory for exporter
RUN mkdir /exporter && chown varnish /exporter

# Copy built binaries
COPY --from=builder /workspace/kube-httpcache .
COPY --from=builder-exporter /workspace/prometheus_varnish_exporter/prometheus_varnish_exporter /exporter/

# Entrypoint
ENTRYPOINT ["/kube-httpcache"]
    