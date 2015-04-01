#!/bin/bash
#
# Author: Moritz Wanzenböck (moritz.wanzenboeck@gmail.com)
#
# Usage: ./addchallenge.sh <challenge-name>
# Script to add a new challenge on this server

DEFAULT_CONFIG_FILE=config.rc
QUICK_LINK_DIR=/challenge/

# $CONFIG_FILE should point at the config file created by setup.sh
if [ -z $CONFIG_FILE ]; then
	CONFIG_FILE=$DEFAULT_CONFIG_FILE
fi

eval $(cat config.rc)

# Challenge needs a name
if [ -z $1 ]; then
	echo "Challenge must have a non-empty name"
	exit 1
fi

CHALLENGE_NAME=$1


# Check if challenge already exists
cd $REPO_DIR
if [ -d $CHALLENGE_NAME".git" ]; then
	echo "Challenge name already in use"
	exit 1
fi

echo "Create directory for challenge '$CHALLENGE_NAME'"
mkdir $CHALLENGE_NAME".git"
cd $CHALLENGE_NAME".git"

# Init git repo without working directory
echo "Let git init the repo"
git --bare init 

rm -rf hooks
ln -s $GIT_HOME/hooks hooks 
chmod a+x $(pwd)/hooks/*
chown -R $GIT_USER:$GIT_GROUP $(pwd)

# Link directory for convinient access
echo "Setting link"
mkdir -p $QUICK_LINK_DIR
ln -s $(pwd) $QUICK_LINK_DIR/$CHALLENGE_NAME".git"
