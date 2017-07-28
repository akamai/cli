#!/bin/bash
function check_version {
	grep "app.Version" ./akamai.go | grep  \"$1\"
	if [[ $? -eq 1 ]]
	then
		echo "app.Version hasn't been updated"
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

GOOS=darwin GOARCH=amd64 go build -o akamai-$1-macamd64 .
GOOS=linux GOARCH=amd64 go build -o akamai-$1-linuxamd64 .
GOOS=linux GOARCH=386 go build -o akamai-$1-linux386 .
GOOS=windows GOARCH=386 go build -o akamai-$1-windows386.exe .
GOOS=windows GOARCH=amd64 go build -o akamai-$1-windowsamd64.exe .
