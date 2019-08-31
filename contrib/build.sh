#!/bin/bash
##########################################################################
# A small script to build a static dn42regsrv image
# using the golang container image
#
# the binary will be built in to the current directory
##########################################################################
RUNTIME=$(which podman || which docker)
echo "Using container runtime: ${RUNTIME}"

# find the source directory
SCRIPTPATH="$(cd "$(dirname "$0")" ; pwd -P)"
SOURCEPATH="$(cd "${SCRIPTPATH}/../"; pwd -P)"
echo "Source is in: ${SOURCEPATH}"

# do the thing
${RUNTIME} run --rm \
           -e CGO_ENABLED=0 \
           -v "${SOURCEPATH}:/go/src/dn42regsrv" \
           -v "${PWD}:/go/bin" \
           -w "/go/src/dn42regsrv" \
           docker.io/golang:1.12 \
           go get

##########################################################################
# end of code
