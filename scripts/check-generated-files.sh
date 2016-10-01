#!/bin/bash

set -euo pipefail

# Ensure all JSON files in koding source tree are formatted correctly,
# so bindata won't generate new versions if e.g. one has formatter
# plugged into their IDE.
find go/src/koding -type f -name '*.json' -exec  ex -sc '%!jq .' -cx {} \;

# Ensure there are no stale, generated files.
#
# NOTE(rjeczalik): For go-bindata it is good to pass fixed -mode and -modtime
# flag values, so the files are no regenerated each time.
# See koding/config/config.go for an example.
go generate koding/...


# Ensure there are no changes in the working tree.
git diff --name-only --exit-code go/src/koding
