#!/bin/bash

# Inputs / outputs
SCAN_DIR=${GOPATH}/pkg/mod
OUT_DIR=licenses

# Fail on error
set -e

# Clean up before we start our work
rm -rf $OUT_DIR/*/

# Get any file in the vendor directory with the word "license" in it.  Note that we'll also keep its path
myLicenses=$(find $SCAN_DIR -type f | grep -i license)
for licensefile in $myLicenses
do
    # make a new license directory matching the same vendor structure
    licensedir=$(dirname $licensefile)
    newlicensedir=$(echo $licensedir | sed "s:$SCAN_DIR:$OUT_DIR:" | sed 's:@[0-9a-zA-Z.\\-]*/:/:' | sed 's:@[0-9a-zA-Z.\\-]*::')
    mkdir -p $newlicensedir
    # And, copy over the license
    cp -f $licensefile $newlicensedir
done

sudo chmod -R 755 licenses
