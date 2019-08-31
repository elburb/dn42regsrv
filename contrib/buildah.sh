#!/bin/bash
##########################################################################
echo "Building dn42regsrv container"

# find the source directory
SCRIPTPATH="$(cd "$(dirname "$0")" ; pwd -P)"
SOURCEPATH="$(cd "${SCRIPTPATH}/../"; pwd -P)"
echo "Source is in: ${SOURCEPATH}"

##########################################################################

DEPS='git'
B=$(which buildah)

# initialise container
c=$(buildah from --name dn42regsrv-working docker.io/debian:buster)

##########################################################################

# install dependencies and initialise directories
$B run $c -- bash <<EOF
apt-get -y update
apt-get -y install --no-install-recommends $DEPS
rm -r /var/lib/apt/lists
EOF

# mount the container
m=$($B mount $c)

# create directories and copy the web app
mkdir "$m/app" "$m/data" "$m/registry" "$m/data/ssh"

# web app
cp -r "$SOURCEPATH/StaticRoot" "$m/data/webapp"

# add the entrypoint.sh script
cp "$SOURCEPATH/contrib/entrypoint.sh" "$m/app"
chmod +x "$m/app"

# build the binary directly in to the container
pushd "$m/app"
"$SOURCEPATH/contrib/build.sh"
popd

# unmount the container
$B unmount $c

# configure
buildah config \
        --author="Simon Marsh" \
        --workingdir='/data/registry' \
        --cmd='/app/entrypoint.sh' \
        $c

##########################################################################
# finally create the image

echo "Committing image ..."
i=$($B commit --squash $c dn42regsrv)

# clean up
$B rm $c

##########################################################################
# end of file
