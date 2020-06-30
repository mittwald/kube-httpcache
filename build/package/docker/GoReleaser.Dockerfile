FROM        debian:stretch-slim

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

WORKDIR     /

RUN         apt-get -qq update && apt-get -qq upgrade \
            && \
            apt-get -qq install \
                debian-archive-keyring \
                curl \
                gnupg \
                apt-transport-https \
            && \
            curl -Ss -L https://packagecloud.io/varnishcache/varnish60lts/gpgkey | apt-key add - \
            && \
            printf "%s\n%s" \
                "deb https://packagecloud.io/varnishcache/varnish60lts/debian/ stretch main" \
                "deb-src https://packagecloud.io/varnishcache/varnish60lts/debian/ stretch main" \
            > "/etc/apt/sources.list.d/varnishcache_varnish60lts.list" \
            && \
            apt-get -qq update && apt-get -qq install varnish \
            && \
            apt-get -qq purge curl gnupg apt-transport-https && \
            apt-get -qq autoremove && apt-get -qq autoclean && \
            rm -rf /var/cache/*

COPY        kube-httpcache .

ENTRYPOINT [ "/kube-httpcache" ]