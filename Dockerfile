FROM ubuntu:vivid

MAINTAINER "Coduno <team@cod.uno>"

RUN apt-get -y install curl git
RUN curl -s https://coduno.github.io/cli/install.sh | bash -s - -y
RUN apt-get -y install gcc
RUN apt-get -y install g++
RUN apt-get -y install python2.7
RUN apt-get update
RUN apt-get -y install openjdk-7-jdk

WORKDIR /run

ENTRYPOINT ["/bin/bash", "-c", "coduno prepare < /dev/null > prepare.log && coduno run --stats"]
