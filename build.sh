#!/bin/bash
GOOS=darwin GOARCH=amd64 go build -o akamai-$1-macamd64 .
GOOS=linux GOARCH=amd64 go build -o akamai-$1-linuxamd64 .
GOOS=linux GOARCH=386 go build -o akamai-$1-linux386 .
GOOS=windows GOARCH=386 go build -o akamai-$1-windows386.exe .
GOOS=windows GOARCH=amd64 go build -o akamai-$1-windowsamd64.exe .

