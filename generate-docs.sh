#!/bin/bash

# Copyright 2018 Crunchy Data Solutions, Inc.
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

if [[ $(git status -s) ]]
then
    echo "The working directory is dirty. Please commit any pending changes."
    exit 1;
fi

echo "Moving to the Hugo subdirectory..."
# Navigate to directory containing Hugo files
cd ${COROOT?}/hugo/

# Generate documents under /docs/
echo "Generating Hugo webpages..."
hugo

# Add all changes and commit, push to GitHub
echo "Updating GitHub repository..."
git add --all && git commit -m "Publishing documentation"

echo "Next Steps: Push your commits to your working repository and submit a pull request."
