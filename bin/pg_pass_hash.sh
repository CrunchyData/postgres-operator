#!/bin/bash

set -ieu

MD5SUM=$(command -v md5sum)
if [ $? -eq 1 ]; then
    echo "md5sum utility not found in your path"
    exit 1
fi

echo ""
echo "This script creates a md5 hash compatible with PostgreSQL database"
echo ""

# Read username
printf "DB Username:"
read username

# Read password
stty -echo
printf "Password:"
read  password
stty echo

printf "\n"

rawhash=$(echo -n "$password$username" | ${MD5SUM} | awk '{print $1}')
md5hash="md5$rawhash"

echo "MD5HASH:$md5hash"
echo ""

echo "Usage Examples:"
echo "CREATE ROLE $username PASSWORD '$md5hash';"
echo "ALTER ROLE $username PASSWORD '$md5hash';"

