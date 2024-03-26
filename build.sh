#!/bin/bash

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
go build