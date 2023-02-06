#!/usr/bin/env bash

# Accept a file path for param yaml.
releaseParams=$1

# Two kinds of operating system can appear in image-tags.
operatingSystems=("ubi8" "ubi9")

# TODO: Eventually, we'll need two kinds of architecture: arm and x86.

# Get the paths to all files that can be automatically updated for a release.
readarray paths < <(yq e -o=j -I=0 '.files[].file' $releaseParams)

echo Evaluating the following files for tag updates:

# For each file path, remove its double quotation marks, then make updates.
for filePath in "${paths[@]//\"/}"; do
  echo -e '\t' $filePath

  # Retrieve the update objects for the current file path.
  readarray updates < <(fp=$filePath yq eval -o=j -I=0 '.files[] | select(.file==env(fp)).updates[]' $releaseParams)

  for update in "${updates[@]}"; do
    # The tag type will dictate find-and-replace strategy.
    tagType=$(echo $update | yq -P '.tag-type')

    # Exit if no tag-type is specified.
    if [ $tagType = null ]; then
      echo $tagType
      echo Missing update tagType in $filePath
      exit 1
    fi

    # Look up the replacement value in release-tags.
    replacementKey=$(echo $update | yq -P '.replacement') 
    replacement=$(cat $releaseParams | yq -e ".release-tags.${replacementKey}")

    # Get the base template for find-and-replace operations.
    base=$(echo $update | yq -P '.base') 
    style=$(echo $update | yq -P '.style')

    # If tagType is image-tag, loop over operating systems to find-and-replace.
    if [ $tagType = 'image-tag' ]; then
      for os in "${operatingSystems[@]//\"}"; do
          if [ $style = 'double-quoted-rhs' ]; then
            find=$base\"$os$(echo $update | yq -P '.regex')\"
            replacement=$base\"$os$replacement\"
          else
            find=$base$os$(echo $update | yq -P '.regex') 
            replacement=$base$os$replacement
          fi
          # Perform the file update.
          sed -i "s|$find|$replacement|" $filePath
      done
    fi

    if [ $tagType = 'version' ]; then
      if [ $style = 'double-quoted-rhs' ]; then
        find=$base\"$(echo $update | yq -P '.regex')\"
        replacement=$base\"$replacement\"
      else
        find=$base$(echo $update | yq -P '.regex')
        replacement=$base$replacement
      fi
      # Perform the file update.
      sed -i "s|$find|$replacement|" $filePath
    fi
  done
done
