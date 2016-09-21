#!/usr/bin/env bash
#
# @author Illarion Kovalchuk <illarion.kovalchuk@gmail.com>
#
# Sample script for exec healthcheck of dns backends for udp protocol

host=$1
port=$2

dig @"$host" -p "$port" +time=1 > /dev/null 2>&1 ; [[ "$?" == "0" ]] && echo -n 1 || echo -n 0