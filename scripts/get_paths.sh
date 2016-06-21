#!/bin/bash

absolute_path() {
  (cd $1 && pwd)
}

scripts_path=$(absolute_path `dirname $0`)

PERSI_ACCEPTANCE_DIR=${PERSI_ACCEPTANCE_DIR:-$(absolute_path $scripts_path/..)}

echo PERSI_ACCEPTANCE_DIR=$PERSI_ACCEPTANCE_DIR
