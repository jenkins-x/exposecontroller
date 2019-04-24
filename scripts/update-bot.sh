#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

# Update other repo's dependencies on jx to use the new version - updates repos as specified at .updatebot.yml
updatebot push-version --kind helm jx $VERSION