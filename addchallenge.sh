#!/bin/bash
#
# Author: Moritz Wanzenböck (moritz.wanzenboeck@gmail.com)
#
# Usage: ./addchallenge.sh <challenge-name>
# Script to add a new challenge on this server

DEFAULT_CONFIG_FILE="config.rc"

# Exit on error
set -e

# $CONFIG_FILE should point at the config file created by setup.sh
if [ -z "$CONFIG_FILE" ]; then
	CONFIG_FILE="$DEFAULT_CONFIG_FILE"
fi

eval $(cat "$CONFIG_FILE")

# Challenge needs a name
if [ -z "$1" ]; then
	echo "Challenge must have a non-empty name"
	exit 1
fi

CHALLENGE_NAME="$1"

# Check if challenge already exists
if [ -d "$REPO_DIR/$CHALLENGE_NAME" ]; then
	echo "Challenge name already in use"
	exit 1
fi

echo "Create directory for challenge '$CHALLENGE_NAME'"
mkdir -p "$REPO_DIR/$CHALLENGE_NAME"

# Init git repo without working directory
echo "Let git init the repo"
git --bare init "$REPO_DIR/$CHALLENGE_NAME"

rm -rf "$REPO_DIR/$CHALLENGE_NAME/hooks"
ln -s "$GIT_HOME/hooks" "$REPO_DIR/$CHALLENGE_NAME/hooks"
chmod -R a+x "$REPO_DIR/$CHALLENGE_NAME/hooks/"
chown -R "$GIT_USER:$GIT_GROUP" "$REPO_DIR/$CHALLENGE_NAME"

# Init git repo for tests
mkdir -p "$GIT_HOME/tests/$CHALLENGE_NAME"
git --bare init "$GIT_HOME/tests/$CHALLENGE_NAME"
chown -R "$GIT_USER:$GIT_GROUP" "$GIT_HOME/tests/$CHALLENGE_NAME"

# Link directory for convinient access
echo "Setting link"
ln -s "$REPO_DIR/$CHALLENGE_NAME" "$GIT_HOME/$CHALLENGE_NAME"
