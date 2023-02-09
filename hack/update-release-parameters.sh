#!/usr/bin/env bash

set -eu

# Accept a file path for param yaml.
releaseParams=$1

# Get target files, templates and parameter replacement values.
updates=$(yq '.updates[]' "$releaseParams")
readarray -t updates < <(jq -c <<< "$updates")

for update in "${updates[@]}"; do
  targetFile=$(yq '.targetFile' <<< "$update")
  templatePath=$(yq '.template' <<< "$update")
  echo -e "Regenerating... $targetFile"
  draft=$(cat "${templatePath//\"/}")

  replacements=$(yq '.replacements[]' <<< "$update")
  readarray -t replacements < <(jq -c <<< "$replacements") 
  for find in "${replacements[@]//\"/}"; do
    find=$(tr -d '\n' <<< "$find")
    replace=$(yq ".release_tags.${find}" "${releaseParams//\"/}")
    draft=$(sed -e "s|{{$find}}|${replace//\"/}|g" <<< "$draft")
  done
  # Write to file.
  echo "$draft" > "${targetFile//\"/}"
done

make generate-crd
