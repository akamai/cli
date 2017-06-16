#!/bin/bash
GOOS=darwin GOARCH=amd64 go build -o akamai-macamd64 .
GOOS=linux GOARCH=amd64 go build -o akamai-linuxamd64 .
GOOS=linux GOARCH=386 go build -o akamai-linux386 .
GOOS=windows GOARCH=386 go build -o akamai-windows386.exe .
GOOS=windows GOARCH=amd64 go build -o akamai-windowsamd64.exe .

