#!/bin/bash

# check_deps.sh
# this script checks for changes to the files Gopkg.lock and Gopkg.toml
# or more specificaly anything matching Gopkg.*
#
# this is intended to be used in your CI tests
#
# on encountering any changes for these files the script runs dep ensure
# with the -dry-run option to check for any conflicts in versions or digests
# which on any exit code > 0 would suggest that action should be taken
# before a pull request can be merged.
set -e

CHANGED_FILES=`git diff --name-only master`

for FILE in $CHANGED_FILES; do
  if [[ "$FILE" =~ ^Gopkg.* ]]; then
    dep ensure -dry-run
    break
  fi
done
