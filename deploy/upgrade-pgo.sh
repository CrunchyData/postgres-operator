#!/bin/bash

# Copyright 2020 Crunchy Data Solutions, Inc.
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

# Get current working directory
DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo ""
echo "Before running the Postgres Operator upgrade script, please ensure you have already updated and"
echo "sourced your user's .bashrc file, as well as your \$PGOROOT\\postgres-operator\\pgo.yaml configuration file."
echo ""
echo "More information can be found in the \"Default Installation - Configure Environment\" section"
echo "of the Postgres Operator Bash installation instructions, located here:"
echo ""
echo "https://crunchydata.github.io/postgres-operator/stable/installation/operator-install/"
echo ""

read -n1 -rsp $'Press any key to continue the upgrade or Ctrl+C to exit...\n'

# Remove the current Operator
$DIR/cleanup.sh

# Set up the defined namespaces for use with the new Operator version
$DIR/setupnamespaces.sh

# Install the correct RBAC
$DIR/install-rbac.sh

# Deploy the new Operator
$DIR/deploy.sh

# Store the current location of the PGO client
MYPGO=`which pgo`
# Store the expected location of the PGO client
BASHPGO="${GOBIN}/pgo"

if [ "$MYPGO" != "$BASHPGO" ]; then

	echo "Current location\(${MYPG}O\) does not match the expected location \(${BASHPGO}\). You will need to manually install the updated Posgres Operator client in your preferred location."

else
	# install the new PGO client
	go install $PGOROOT/pgo/pgo.go
	cp $GOBIN/pgo $PGOROOT/bin/pgo
fi

# Final instructions
NEWLINE=$'\n'
echo ""
echo ""
echo "Postgres Operator upgrade has completed!"
echo ""
echo "Please note it may take a few minutes for the deployment to complete,"
echo ""
echo "and you will need to use the setip function to update your Apiserver URL once the Operator is ready."
echo ""
