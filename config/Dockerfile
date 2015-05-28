FROM fedora:21

MAINTAINER "Moritz Wanzenböck <moritz.wanzenboeck@cod.uno>"

# Install all default packages
RUN yum -y install curl tar git golang make

# Setup go
RUN mkdir -p /go
ENV GOPATH /go

# Setup for coduno client
RUN mkdir -p /coduno
RUN git clone http://github.com/coduno/cli.git /coduno
RUN cd /coduno; make

# Set working directory
WORKDIR /app

ENTRYPOINT ["/bin/bash", "-c", "/coduno/coduno prepare > /dev/null && /coduno/coduno run"]
