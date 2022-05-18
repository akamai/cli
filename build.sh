#!/bin/bash
# Creates unsigned binaries for macOS (64bit), and Linux/Windows (32 and 64bit) 
function get_version {
	ver=$(sed -En "s/^.*Version = \"(.*)\"/\1/p" pkg/version/version.go)
}

get_version

mkdir -p build/"$ver"
GOOS=darwin GOARCH=amd64 go build -o build/"$ver"/akamai-v"$ver"-macamd64 ./cli/main.go
shasum -a 256 build/"$ver"/akamai-v"$ver"-macamd64 | awk '{print $1}' > build/"$ver"/akamai-v"$ver"-macamd64.sig
GOOS=darwin GOARCH=arm64 go build -o build/"$ver"/akamai-v"$ver"-macarm64 ./cli/main.go
shasum -a 256 build/"$ver"/akamai-v"$ver"-macarm64 | awk '{print $1}' > build/"$ver"/akamai-v"$ver"-macarm64.sig
GOOS=linux GOARCH=amd64 go build -o build/"$ver"/akamai-v"$ver"-linuxamd64 ./cli/main.go
shasum -a 256 build/"$ver"/akamai-v"$ver"-linuxamd64 | awk '{print $1}' > build/"$ver"/akamai-v"$ver"-linuxamd64.sig
GOOS=linux GOARCH=386 go build -o build/"$ver"/akamai-v"$ver"-linux386 ./cli/main.go
shasum -a 256 build/"$ver"/akamai-v"$ver"-linux386 | awk '{print $1}' > build/"$ver"/akamai-v"$ver"-linux386.sig
GOOS=windows GOARCH=386 go build -o build/"$ver"/akamai-v"$ver"-windows386.exe ./cli/main.go
shasum -a 256 build/"$ver"/akamai-v"$ver"-windows386.exe | awk '{print $1}' > build/"$ver"/akamai-v"$ver"-windows386.exe.sig
GOOS=windows GOARCH=amd64 go build -o build/"$ver"/akamai-v"$ver"-windowsamd64.exe ./cli/main.go
shasum -a 256 build/"$ver"/akamai-v"$ver"-windowsamd64.exe | awk '{print $1}' > build/"$ver"/akamai-v"$ver"-windowsamd64.exe.sig
