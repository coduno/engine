#!/bin/bash
#
# Author: Moritz Wanzenböck (moritz.wanzenboeck@gmail.com)
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
#   |   |-pkg
#   |   \-src
#   |      \-testrun
#   |         \-testrun.go
#   |-hooks
#   |   |-post-receive
#   |   \-update
#   |-git-shell-commands
#   |   \-no-interactive-login
#   |-config
#   |   |-Dockerfile
#   |   \-secret.json
#   |-user
#       |-<example user>
#       |    \-<repo copy>
#       \-<example user2>
#

PACKAGES="openssh-server git docker.io"
GIT_USER="git"
GIT_GROUP="git"
GIT_SHELL="/usr/bin/git-shell"
DOCKER_TAG="coduno/base"
DEFAULT_REPO_DIR="/opt/coduno/engine/repo/"
DEFAULT_GIT_HOME="/opt/coduno/engine/git/"
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
echo "Install required packages"
apt-get -y install $PACKAGES > /dev/null 2>&1 || \
yum -y install $PACKAGES > /dev/null 2>&1


# Add git user, if neccessary
echo "Setting up user '$GIT_USER'"
if id -u "$GIT_USER" > /dev/null 2>&1 ; then
	# User exists, point $GIT_HOME to right directory
	GIT_HOME="$(eval echo ~$GIT_USER)"
	GIT_GROUP="$(id -g -n $GIT_USER)"
	echo "User '$GIT_USER' already exists, setting \$GIT_HOME to $GIT_HOME"
else
	echo "Adding new user '$GIT_USER'"
	useradd -d "$GIT_HOME" -s "$GIT_SHELL" "$GIT_USER"
	GIT_GROUP="$(id -g -n $GIT_USER)"
	echo "Create home directory '$GIT_HOME'"
	mkdir -p "$GIT_HOME"
	chown "$GIT_USER:$GIT_GROUP" "$GIT_HOME"
fi

# Add .ssh/authorized_keys file to $GIT_USER
echo "Ensure there is an authorized_keys file for user '$GIT_USER' at '$GIT_HOME'"
mkdir -p "$GIT_HOME/.ssh"
touch "$GIT_HOME/.ssh/authorized_keys"
chown -R "$GIT_USER:$GIT_GROUP" "$GIT_HOME/.ssh"

# Set greeting for the interactive shell
echo "Set greeting message for user git"
mkdir -p "$GIT_HOME/git-shell-commands"
cp "./config/greeting" "$GIT_HOME/git-shell-commands/no-interactive-login"
chmod a+x "$GIT_HOME/git-shell-commands/no-interactive-login"

# Copy reroute-user script for access-control to allowed commands
echo "Copy reroute-user to allowed commands"
cp "./config/reroute-user" "$GIT_HOME/git-shell-commands/reroute-user"
chown -R "$GIT_USER:$GIT_GROUP" "$GIT_HOME/git-shell-commands"
chmod a+x "$GIT_HOME/git-shell-commands/reroute-user"

# Prevent SSHd from printing MOTD and Last Login
touch "$GIT_HOME/.hushlogin"
chmod 0444 "$GIT_HOME/.hushlogin"

# Copy hooks to git directory
echo "Copy hooks to '$GIT_HOME'"
mkdir -p "$GIT_HOME/hooks"
cp "./hooks/post-receive" "./hooks/update" "$GIT_HOME/hooks"
chown -R "$GIT_USER:$GIT_GROUP" "$GIT_HOME/hooks"

# Set up go path and copy testrun source into into
echo "Copy go source files"
mkdir -p "$GIT_HOME/go/src"
cp -r gosrc/* "$GIT_HOME/go/src"
chown -R "$GIT_USER:$GIT_GROUP" "$GIT_HOME/go"

# Set GOPATH and build testrun
echo "Build testrun command"
echo "export GOPATH=$GIT_HOME/go" > "$GIT_HOME/.bashrc"
chown "$GIT_USER:$GIT_GROUP" "$GIT_HOME/.profile"
su -s "/bin/bash" - git -c "go get testrun && go build testrun"

# Copy config files
echo "Copy config files"
mkdir -p "$GIT_HOME/config"
cp ./config/* "$GIT_HOME/config"
chown -R "$GIT_USER:$GIT_GROUP" "$GIT_HOME/config"
chmod -R 555 "$GIT_HOME/config"

# Ensure $REPO_DIR exists
echo "Ensure that the repo directory exists at '$REPO_DIR'"
mkdir -p "$REPO_DIR"

# Ensure git has write access to $REPO_DIR
echo "Change owner of '$REPO_DIR' to '$GIT_USER'"
chown -R "$GIT_USER:$GIT_GROUP" "$REPO_DIR"

# Ensure docker is running and create image '$DOCKER_TAG'
echo "Ensure that docker is running and create image '$DOCKER_TAG'"
service docker start
docker build -t "$DOCKER_TAG" "./docker/base/"

# Write config
echo "export REPO_DIR=$REPO_DIR" > "$CONFIG_FILE"
echo "export GIT_USER=$GIT_USER" >> "$CONFIG_FILE"
echo "export GIT_HOME=$GIT_HOME" >> "$CONFIG_FILE"
echo "export GIT_GROUP=$GIT_GROUP" >> "$CONFIG_FILE"
echo "export GIT_SHELL=$GIT_SHELL" >> "$CONFIG_FILE"
