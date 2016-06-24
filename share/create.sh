#!/usr/env bash
curl -XPOST -v "http://localhost:8888/servers/test" --data '
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
