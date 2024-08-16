#!/bin/bash

# This file is for CodeQl

# Define a function to handle errors
handle_error() {
    echo "An error occurred. Exiting..."
    exit 1
}

# Set the error trap
trap 'handle_error' ERR

echo "Fetching dependencies"
go mod tidy

if command -v templ &> /dev/null; then
    echo "templ command found!"
else
    echo "templ command not found. Installing..."
    go install github.com/a-h/templ/cmd/templ@latest
    echo "templ installed successfully!"
fi

echo "Generating from templ files"
templ generate
echo "Building the app"
go build -v -race ./...

# If the build is successful, execute the following code
echo "App built successfully!"