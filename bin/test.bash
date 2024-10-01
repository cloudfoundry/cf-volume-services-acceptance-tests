#!/bin/bash

set -eu
set -o pipefail

if [[ "${CONFIG:-empty}" == "empty" ]]; then
    echo "Provide a CONFIG file"
    exit 1
fi
# shellcheck disable=SC2068
# Double-quoting array expansion here causes ginkgo to fail
go run github.com/onsi/ginkgo/v2/ginkgo ${@}
