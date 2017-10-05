#!/bin/bash
# Creates unsigned binaries for macOS (64bit), and Linux/Windows (32 and 64bit) 
function check_version {
	grep "VERSION" ./akamai.go | grep  \"$1\"
	if [[ $? -eq 1 ]]
	then
		echo "VERSION hasn't been updated"
		exit 1
	fi
}

if [[ -z "$1" ]]
then
	echo "Version not supplied."
	echo "Usage: build.sh <version>"
	exit 1
fi

check_version $1

mkdir -p build

GOOS=darwin GOARCH=amd64 go build -o build/akamai-$1-macamd64 .
shasum -a 256 build/akamai-$1-macamd64 | awk '{print $1}' > build/akamai-$1-macamd64.sig
GOOS=linux GOARCH=amd64 go build -o build/akamai-$1-linuxamd64 .
shasum -a 256 build/akamai-$1-linuxamd64 | awk '{print $1}' > build/akamai-$1-linuxamd64.sig
GOOS=linux GOARCH=386 go build -o build/akamai-$1-linux386 .
shasum -a 256 build/akamai-$1-linux386 | awk '{print $1}' > build/akamai-$1-linux386.sig
GOOS=windows GOARCH=386 go build -o build/akamai-$1-windows386.exe .
shasum -a 256 build/akamai-$1-windows386.exe | awk '{print $1}' > build/akamai-$1-windows386.exe.sig
GOOS=windows GOARCH=amd64 go build -o build/akamai-$1-windowsamd64.exe .
shasum -a 256 build/akamai-$1-windowsamd64.exe | awk '{print $1}' > build/akamai-$1-windowsamd64.exe.sig
