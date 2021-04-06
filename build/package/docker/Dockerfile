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


# build varnish and vmods
FROM        ubuntu:18.04 AS varnish-builder
ENV         DEBIAN_FRONTEND=noninteractive DEBCONF_NONINTERACTIVE_SEEN=true
RUN         apt-get update
RUN         apt-get install -y curl automake libtool python-docutils python3 python3-sphinx pkg-config libpcre++-dev libedit-dev vim
COPY        varnish /root/

# varnish 6.0.6
RUN         (cd /root; tar zxf varnish-cache-varnish-6.5.1.tar.gz)
WORKDIR     /root/varnish-cache-varnish-6.5.1
RUN         ./autogen.sh
RUN         ./configure --prefix=/opt/varnish
RUN         make
RUN         make install
ENV         PKG_CONFIG_PATH=/opt/varnish/lib/pkgconfig

# varnish-modules
RUN         (cd /root; tar zxf varnish-modules-6.5.tar.gz)
WORKDIR     /root/varnish-modules-6.5
RUN         ./bootstrap
RUN         ./configure --prefix=/opt/varnish
RUN         make
RUN         make install

# final image
FROM        ubuntu:18.04 AS final

WORKDIR     /
RUN         apt-get update && apt-get install -y gcc libedit2 libpcre++ && rm -rf /var/lib/apt/lists/*
COPY        --from=varnish-builder /opt/varnish /opt/varnish/
COPY        --from=app-builder /workspace/kube-httpcache .
COPY        ./prometheus_varnish_exporter-1.5.2.linux-amd64/prometheus_varnish_exporter /usr/bin
ENV         PATH=/opt/varnish/bin:$PATH

ENTRYPOINT  [ "/kube-httpcache" ]
