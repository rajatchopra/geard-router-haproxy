FROM fedora
RUN yum clean metadata && yum update -y --exclude='rhc*,node*' && \
yum -y install wget && \
wget http://haproxy.1wt.eu/download/1.5/src/snapshot/haproxy-ss-LATEST.tar.gz  && \
tar xvzf haproxy-ss-LATEST.tar.gz && \
groupadd haproxy && \
useradd -g haproxy haproxy && \
yum -y install gcc make openssl-devel pcre-devel socat && \
cd haproxy-ss-* && make TARGET=linux2628 CPU=native USE_PCRE=1 USE_OPENSSL=1 USE_ZLIB=1 && make install && \
cd .. && rm -rf haproxy-ss-* && \
yum -y remove gcc && \
mkdir -p /var/lib/haproxy/{conf,run,bin,log} && touch /var/lib/haproxy/conf/{host_be.map,host_be_ws.map,host_be_ressl.map,host_be_sni.map,haproxy.config}
VOLUME /var/lib/
EXPOSE 80
CMD ["/bin/bash -c /var/lib/haproxy/bin/signal_router"]
