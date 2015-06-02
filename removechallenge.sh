#!/bin/bash
#
# Author: Moritz Wanzenböck (moritz.wanzenboeck@gmail.com)
#
# Script to remove unused or outdated challenges

RUN_DIR=$(pwd)
DEFAULT_CONFIG_FILE=config.rc
QUICK_LINK_DIR=/challenge/

# Exit on error
set -e

# $CONFIG_FILE should point at the config file created by setup.sh
if [ -z $CONFIG_FILE ]; then
	CONFIG_FILE=$DEFAULT_CONFIG_FILE
fi

eval $(cat config.rc)

# Challenge needs a name
if [ -z $1 ]; then
	echo "Cannot remove unspecified challenge"
	exit 1
fi

CHALLENGE_NAME=$1

echo "Remove git repo"
rm -rf $REPO_DIR/$CHALLENGE_NAME".git" > /dev/null 2>&1
