#!/usr/bin/env bash
#
# Create new server.
# Format should correspond to [servers] entry in TOML file
#
curl -XPOST "http://localhost:8888/servers/$1" --data '
{
    "bind":"localhost:3001",

    "healthcheck": {
        "kind": "ping",
        "interval": "2s",
        "timeout": "1s"
    },

    "discovery": {
        "kind": "static",
        "static_list": ["localhost:8000"]
    }
}
'
