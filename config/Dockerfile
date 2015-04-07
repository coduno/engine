FROM fedora:21

MAINTAINER "Moritz Wanzenböck <moritz.wanzenboeck@cod.uno>"

# Setup for go
RUN mkdir -v /go
ENV GOPATH /go


RUN yum -y install curl tar git golang
#RUN curl 'https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz' | tar --exclude='go/doc' --exclude='go/blog' -xzf - -C /usr/local
#ENV PATH $PATH:/usr/local/go/bin


# Get app into the container
ADD ./src /app
WORKDIR /app
