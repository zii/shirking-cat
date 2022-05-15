#!/bin/bash

set -e

# run one/many
for arg in "$@"
do
  if [[ $arg == 1 ]]; then
    echo "pull&restart..."
    git pull
    go build -o ./zdm zdm
    supervisorctl restart zdm
  else
    echo unknown argument: $arg
  fi
done