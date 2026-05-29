#!/usr/bin/env bash
#
# @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
#
# Sample script for exec healthcheck
# For more info see sample gobetween.toml
#
# gobetween expects (by default):
#  - singe character '1' in output (without newline and quotes) - if healcheck was successfull,
#  - singe character '0' in output - if healcheck failed
#  - on any other output or script error - no change to backend status will be applied
# It may be overriden in configuration file.
#
# first and second arguments to the script is host and port, it will be called as:
# yourcmd <host> <port>
#

host=$1
port=$2

if [[ "$host:$port" = "localhost:8000" ]]; then 
    echo -n 1
else
    echo -n 0
fi
