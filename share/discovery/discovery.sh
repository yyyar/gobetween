#!/usr/bin/env bash
#
# @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
#
# Sample script for exec discovery.
# Should write to stdout string in the same format as
# in static discovery separated with newlines.
# No newline needed after the content.
#
# For more info see sample gobetween.toml
#

echo localhost:8000 weight=1
echo localhost:8001 weight=2
