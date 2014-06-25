FROM fedora
RUN yum clean metadata && yum update -y --exclude='rhc*,node*' && \
yum -y install wget && \
wget http://haproxy.1wt.eu/download/1.5/src/snapshot/haproxy-ss-LATEST.tar.gz  && \
tar xvzf haproxy-ss-LATEST.tar.gz && \
groupadd haproxy && \
useradd -g haproxy haproxy && \
yum -y install gcc make openssl-devel pcre-devel socat golang git && \
cd haproxy-ss-* && make TARGET=linux2628 CPU=native USE_PCRE=1 USE_OPENSSL=1 USE_ZLIB=1 && make install && \
cd .. && rm -rf haproxy-ss-* && \
export GOPATH=/root/ && \
(go get github.com/openshift/geard-router-haproxy || true) && \
cd $GOPATH/src/github.com/openshift/geard-router-haproxy && ./build && \
mkdir -p /usr/bin && \
mkdir -p /var/lib/haproxy/{conf,run,bin,log} && \
cp -f geard-router-haproxy-writeconfig geard-router-haproxy-sighandler /usr/bin/ && \
cp -f haproxy_template.conf default_pub_keys.pem /var/lib/haproxy/conf/ && \
yum -y remove gcc golang git && \
touch /var/lib/haproxy/conf/{host_be.map,host_be_ws.map,host_be_ressl.map,host_be_sni.map,haproxy.config}
VOLUME /var/lib/containers
EXPOSE 80
CMD ["/bin/bash -c /usr/bin/geard-router-haproxy-sighandler"]
