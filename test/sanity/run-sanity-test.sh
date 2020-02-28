#!/bin/bash

# Copyright 2019 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

set -euo pipefail

if [[ -z "${NODEID:-}" ]]; then
  echo "NODEID is not defined"
  exit 1
fi

rm -rf /tmp/csi*

readonly ENDPOINT="unix:///tmp/csi.sock"
./azurediskplugin --endpoint "$ENDPOINT" --nodeid "$NODEID" -v=5 &

echo "Begin to run sanity test..."
csi-sanity --ginkgo.v --ginkgo.noColor --csi.endpoint="$ENDPOINT" --ginkgo.skip="should work"
