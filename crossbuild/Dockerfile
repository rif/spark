FROM ubuntu:latest
MAINTAINER Radu Fericean radu@fericean.ro
# Packaged dependencies
RUN	apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -yq \
automake \
build-essential \
curl \
ca-certificates \
git \
mercurial \
upx \
--no-install-recommends
# Install Go
RUN	curl -s https://storage.googleapis.com/golang/go1.4.1.linux-amd64.tar.gz | tar -v -C /usr/local -xz
ENV	PATH /usr/local/go/bin:$PATH
ENV	GOPATH /go
RUN	cd /usr/local/go/src && ./make.bash --no-clean 2>&1
# Compile Go for cross compilation
ENV	DOCKER_CROSSPLATFORMS linux/386 linux/arm darwin/amd64 darwin/386 windows/386 windows/amd64
# (set an explicit GOARM of 5 for maximum compatibility)
ENV	GOARM 5
RUN	cd /usr/local/go/src && bash -xc 'for platform in $DOCKER_CROSSPLATFORMS; do GOOS=${platform%/*} GOARCH=${platform##*/} ./make.bash --no-clean 2>&1; done'
