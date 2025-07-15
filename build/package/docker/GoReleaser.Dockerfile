ARG DEBIAN_ARCH=amd64
FROM        ${DEBIAN_ARCH}/debian:bookworm-slim

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

WORKDIR     /

# varnish
RUN         apt-get -qq update && apt-get -qq upgrade && apt-get -qq install curl && \
            curl -s https://packagecloud.io/install/repositories/varnishcache/varnish76/script.deb.sh | bash && \
            apt-get -qq update && apt-get -qq install varnish && \
            apt-get -qq purge curl gnupg && \
            apt-get -qq autoremove && apt-get -qq autoclean && \
            rm -rf /var/cache/* && rm -rf /var/lib/apt/lists/*

RUN         mkdir /exporter && \
            chown varnish /exporter

# exporter
ARG ARCH=amd64
ENV         ARCH="${ARCH}"
ENV         EXPORTER_VERSION="v1.7.0"
ADD         --chown=varnish https://github.com/leontappe/prometheus_varnish_exporter/releases/download/${EXPORTER_VERSION}/prometheus_varnish_exporter-${EXPORTER_VERSION}.linux-${ARCH}.tar.gz /tmp

RUN         cd /exporter && \
            tar -xzf /tmp/prometheus_varnish_exporter-${EXPORTER_VERSION}.linux-${ARCH}.tar.gz && \
            ln -sf /exporter/prometheus_varnish_exporter-${EXPORTER_VERSION}.linux-${ARCH}/prometheus_varnish_exporter prometheus_varnish_exporter

COPY        kube-httpcache .

ENTRYPOINT [ "/kube-httpcache" ]