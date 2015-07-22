#!/bin/bash
#
# Authors: Moritz Wanzenböck <moritz.wanzenboeck@cod.uno>
#          Lorenz Leutgeb <lorenz.leutgeb@cod.uno>
#
# Script to set up a debian/ubuntu server to be used as a docker engine
#
# Filesystem layout:
# $REPO_DIR
#   |-<example challenge>
#   |   |-<git stuff>
#   |   |-hooks --> $GIT_DIR/hooks
#
# $GIT_HOME
#   |-.ssh
#   |   \-authorized_keys
#   |-go
#   |   |-bin
#   |   |  \-piper
#   |   |-pkg
#   |   \-src
#   |-hooks
#   |   \-post-receive
#   |-git-shell-commands
#   |   \-no-interactive-login
#   |-config
#   |   \-secret.json
#   |-user
#       |-<example user>
#       |    \-<repo copy>
#       \-<example user2>
#

PACKAGES="openssh-server git docker"
GIT_USER="git"
GIT_GROUP="git"
GIT_SHELL="/usr/bin/git-shell"
DOCKER_TAG="coduno/base"
DEFAULT_REPO_DIR="/opt/coduno/engine/repo"
DEFAULT_GIT_HOME="/opt/coduno/engine/git"
DEFAULT_CONFIG_FILE="config.rc"

# Exit on error
set -e

# $CONFIG_FILE points at the file used to store the created config
if [ -z "$CONFIG_FILE" ]; then
	CONFIG_FILE="$DEFAULT_CONFIG_FILE"
fi

# $REPO_DIR points to the directory that is used to store repos
if [ -z "$REPO_DIR" ]; then
	REPO_DIR="$DEFAULT_REPO_DIR"
fi

# $GIT_HOME points to the home directory of the git user
if [ -z "$GIT_HOME" ]; then
	GIT_HOME="$DEFAULT_GIT_HOME"
fi

# Check for root privileges
if [[ "$EUID" -ne 0 ]]; then
	echo "This script must be run with root privileges" 1>&2
	exit 1
fi

# Install all required packages
apt-get -y install $PACKAGES 2> /dev/null || \
yum -y install $PACKAGES 2> /dev/null || \
dnf -y install $PACKAGES

if ! which go
then
	curl https://storage.googleapis.com/golang/go1.4.2.linux-amd64.tar.gz | tar -C /usr/local -xzv
fi
go version

# Add git user, if neccessary
if id -u "$GIT_USER"
then
	# user exists, so just repoint $GIT_HOME
	GIT_HOME="$(eval echo ~$GIT_USER)"
	echo "\$GIT_HOME -> $GIT_HOME"
else
	useradd --home "$GIT_HOME" --create-home --shell "$GIT_SHELL" "$GIT_USER"
fi
GIT_GROUP="$(id -g -n $GIT_USER)"

# .ssh/authorized_keys
mkdir -vp "$GIT_HOME/.ssh"
touch "$GIT_HOME/.ssh/authorized_keys"
chown -vR "$GIT_USER:$GIT_GROUP" "$GIT_HOME/.ssh"

# greeting
mkdir -vp "$GIT_HOME/git-shell-commands"
cp -vf "./config/greeting" "$GIT_HOME/git-shell-commands/no-interactive-login"
chmod -v a+x "$GIT_HOME/git-shell-commands/no-interactive-login"

# reroute-user
cp -vf "./config/reroute-user" "$GIT_HOME/git-shell-commands/reroute-user"
chown -vR "$GIT_USER:$GIT_GROUP" "$GIT_HOME/git-shell-commands"
chmod -v a+x "$GIT_HOME/git-shell-commands/reroute-user"

# Prevent SSHd from printing MOTD and Last Login
touch "$GIT_HOME/.hushlogin"
chmod -v 0444 "$GIT_HOME/.hushlogin"

# Copy hook to git directory
mkdir -vp "$GIT_HOME/hooks"
cp -vf "./hooks/post-receive" "$GIT_HOME/hooks"
chown -vR "$GIT_USER:$GIT_GROUP" "$GIT_HOME/hooks"

# Set GOPATH and install piper
echo "export GOPATH=$GIT_HOME/go" > "$GIT_HOME/.bashrc"
echo "export PATH=\$PATH:\$PATH:/usr/local/go/bin:\$GOPATH/bin" >> "$GIT_HOME/.bashrc"
chown "$GIT_USER:$GIT_GROUP" "$GIT_HOME/.bashrc"
su -s "$SHELL" - git -c "go get github.com/coduno/piper"

# Copy config files
mkdir -vp "$GIT_HOME/config"
cp -vf ./config/* "$GIT_HOME/config"
chown -vR "$GIT_USER:$GIT_GROUP" "$GIT_HOME/config"
chmod -vR 555 "$GIT_HOME/config"

# Ensure $REPO_DIR exists
mkdir -vp "$REPO_DIR"

# Ensure git has write access to $REPO_DIR
chown -vR "$GIT_USER:$GIT_GROUP" "$REPO_DIR"

# Ensure docker is running and create image '$DOCKER_TAG'
service docker start
docker build -t "$DOCKER_TAG" "./docker/base/"

# Write config
echo "export REPO_DIR=$REPO_DIR" > "$CONFIG_FILE"
echo "export GIT_USER=$GIT_USER" >> "$CONFIG_FILE"
echo "export GIT_HOME=$GIT_HOME" >> "$CONFIG_FILE"
echo "export GIT_GROUP=$GIT_GROUP" >> "$CONFIG_FILE"
echo "export GIT_SHELL=$GIT_SHELL" >> "$CONFIG_FILE"
