#!/bin/sh -e

cp ${SNAP}/config/gobetween.toml ${SNAP_COMMON}/gobetween.toml

echo '#/usr/bin/env bash

${SNAP}/bin/gobetween -c ${SNAP_COMMON}/gobetween.toml
' >> ${SNAP_DATA}/gobetween.sh

chmod +rwx ${SNAP_DATA}/gobetween.sh
