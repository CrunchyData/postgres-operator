#!/bin/bash

# Copyright 2021 Crunchy Data Solutions, Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# Inputs / outputs
SCAN_DIR=${GOPATH:-~/go}/pkg/mod
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
sudo chmod 0644 licenses/LICENSE.txt
