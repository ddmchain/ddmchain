#!/bin/sh

set -e

if [ ! -f "path/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/path/_workspace"
root="$PWD"
ddmdir="$workspace/src/github.com/ddmchain"
if [ ! -L "$ddmdir/go-ddmchain" ]; then
    mkdir -p "$ddmdir"
    cd "$ddmdir"
    ln -s ../../../../../. go-ddmchain
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$ddmdir/go-ddmchain"
PWD="$ddmdir/go-ddmchain"

# Launch the arguments with the configured environment.
exec "$@"
