#!/usr/bin/env sh

set -ex

# build directory
bd="build"
rm -rf "$bd/"*

GGOARCH=amd64 GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -o build/
# tar -czvf "$bd/dict_Linux_x86_64.tar.gz"  -C "$bd" "dict"
# rm "$bd/lains"

GOARCH=arm64 GOOS=linux CGO_ENABLED=0 go build -ldflags "-s -w" -o build/dict_arm
# tar -czvf "$bd/dict_Linux_aarch64.tar.gz" -C "$bd" "dict"
# rm "$bd/dict"
#
# GOOS=windows GOARCH=amd64 CGO_ENABLED=0  go build -ldflags "-s -w" -o build/
# zip -j "$bd/dict_windows_x86_64.zip" "$bd/dict.exe"
# rm "$bd/dict.exe"
