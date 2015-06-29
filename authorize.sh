#!/bin/bash
#
# Author: Lorenz Leutgeb <lorenz.leutgeb@cod.uno>
#
# Usage: ./authorize.sh <user-name>
# Script to add the public key of a GitHub user to the
# list of authorized keys for git on this server

DEFAULT_CONFIG_FILE="config.rc"

# Exit on error
set -e

# $CONFIG_FILE should point at the config file created by setup.sh
if [ -z "$CONFIG_FILE" ]; then
	CONFIG_FILE="$DEFAULT_CONFIG_FILE"
fi

eval $(cat "$CONFIG_FILE")

# User needs a name
if [ -z "$1" ]; then
	echo "User must have a non-empty name"
	exit 1
fi

USER_NAME="$1"

curl -w "\n" "https://github.com/${USER_NAME}.keys" | while read key
do
	[ "" != "$key" ] && echo "environment=\"GITHUB_USERNAME=$USER_NAME\" $key" | tee -a "$GIT_HOME/.ssh/authorized_keys"
done
