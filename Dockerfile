FROM        centos:7

LABEL       MAINTAINER="Martin Helmich <m.helmich@mittwald.de>"

RUN         yum install -y https://dl.fedoraproject.org/pub/epel/epel-release-latest-7.noarch.rpm && \
            yum install -y pygpgme yum-utils

COPY        varnish.repo //etc/yum.repos.d/varnishcache_varnish60.repo

RUN         yum -q makecache -y --disablerepo='*' --enablerepo='varnishcache_varnish60'
RUN         yum install -y varnish

COPY        kube-httpcache /usr/bin/kube-httpcache

ENTRYPOINT [ "/usr/bin/kube-httpcache" ]