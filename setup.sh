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
#   |   \-Dockerfile
#
# $GIT_HOME
#   |-.ssh
#   |   \-authorized_keys
#   |-hooks
#   |   |-post-receive
#   |   \-update
#   |-git-shell-commands
#   |   \-no-interactive-login
#   |-Dockerfile.default
#   |-runTest.py
#

PACKAGES="openssh-server git docker.io"
GIT_USER=git
GIT_GROUP=git
GIT_SHELL=/usr/bin/git-shell
DEFAULT_REPO_DIR=/opt/coduno/engine/repo/
DEFAULT_GIT_HOME=/opt/coduno/engine/git/
DEFAULT_CONFIG_FILE=config.rc

# $CONFIG_FILE points at the file used to store the created config
if [ -z $CONFIG_FILE ]; then
	CONFIG_FILE=$DEFAULT_CONFIG_FILE
fi

# $REPO_DIR points to the directory that is used to store repos
if [ -z $REPO_DIR ]; then
	REPO_DIR=$DEFAULT_REPO_DIR
fi

# $GIT_HOME points to the home directory of the git user
if [ -z $GIT_HOME ]; then
	GIT_HOME=$DEFAULT_GIT_HOME
fi

# Check for root privileges
if [[ $EUID -ne 0 ]]; then
	echo "This script must be run with root privileges" 1>&2
	exit 1
fi

# Install all required packages
echo "Install required packages"
apt-get -y install $PACKAGES > /dev/null


# Add git user, if neccessary
echo "Setting up user '$GIT_USER'"
if id -u $GIT_USER > /dev/null 2>&1 ; then 
	# User exists, point $GIT_HOME to right directory
	GIT_HOME=$(eval echo ~$GIT_USER)
	GIT_GROUP=$(id -g -n $GIT_USER)
	echo "User '$GIT_USER' already exists, setting \$GIT_HOME to $GIT_HOME"
else
	echo "Adding new user '$GIT_USER'"
	useradd -d $GIT_HOME -s $GIT_SHELL $GIT_USER
	GIT_GROUP=$(id -g -n $GIT_USER)
	echo "Create home directory '$GIT_HOME'"
	mkdir -p $GIT_HOME
	chown $GIT_USER:$GIT_GROUP $GIT_HOME
fi

# Add .ssh/authorized_keys file to $GIT_USER
echo "Ensure there is an authorized_keys file for user '$GIT_USER' at '$GIT_HOME'"
mkdir -p $GIT_HOME"/.ssh"
touch $GIT_HOME"/.ssh/authorized_keys"
chown -R $GIT_USER:$GIT_GROUP $GIT_HOME"/.ssh"

# Set greeting for the interactive shell
echo "Set greeting message for user git"
mkdir -p $GIT_HOME"git-shell-commands"
cp ./config/greeting $GIT_HOME"git-shell-commands/no-interactive-login"
chown -R $GIT_USER:$GIT_GROUP $GIT_HOME"git-shell-commands"
chmod a+x $GIT_HOME"git-shell-commands/no-interactive-login"

# Prevent SSHd from printing MOTD and Last Login
touch $GIT_HOME"/.hushlogin"
chmod 0444 $GIT_HOME"/.hushlogin"

# Copy hooks to git directory
echo "Copy hooks to '$GIT_HOME'"
mkdir -p $GIT_HOME/hooks
cp ./hooks/post-receive ./hooks/update $GIT_HOME/hooks
chown -R $GIT_USER:$GIT_GROUP $GIT_HOME"/hooks"

# Copy default Dockerfile to $GIT_HOME
echo "Copy default Dockerfile and runTest.py to '$GIT_HOME'"
cp ./config/Dockerfile $GIT_HOME"/Dockerfile.default"

# Copy runTest.py to $GIT_HOME
cp ./runTest.py $GIT_HOME"/runTest.py"
chmod 555 $GIT_HOME"/runTest.py"

# Ensure $REPO_DIR exists
echo "Ensure that the repo directory exists at '$REPO_DIR'"
mkdir -p $REPO_DIR

# Ensure git has write access to $REPO_DIR
echo "Change owner of '$REPO_DIR' to '$GIT_USER'"
chown -R $GIT_USER:$GIT_GROUP $REPO_DIR

# Write config
echo "export REPO_DIR=$REPO_DIR" > $CONFIG_FILE
echo "export GIT_USER=$GIT_USER" >> $CONFIG_FILE
echo "export GIT_HOME=$GIT_HOME" >> $CONFIG_FILE
echo "export GIT_GROUP=$GIT_GROUP" >> $CONFIG_FILE
echo "export GIT_SHELL=$GIT_SHELL" >> $CONFIG_FILE
