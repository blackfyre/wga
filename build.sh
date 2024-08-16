#!/bin/bash

# This file is for CodeQl

# Define a function to handle errors
handle_error() {
    echo "An error occurred. Exiting..."
    exit 1
}

# Set the error trap
trap 'handle_error' ERR

if command -v templ &> /dev/null; then
    echo "templ command found!"
else
    echo "templ command not found. Installing..."
    go install github.com/a-h/templ/cmd/templ@latest
    echo "templ installed successfully!"
fi

# determine go path and set it
export GO_PATH=$(which go)

# templ is at $GO_PATH/../packages/bin/templ
export TEMPL_PATH=$(dirname $GO_PATH)/../packages/bin/templ

# Run templ to generate the code
echo "Generating code"
$TEMPL_PATH generate

echo "Fetching dependencies"
go mod tidy

echo "Building the app"
go build -v -race ./...

# If the build is successful, execute the following code
echo "App built successfully!"