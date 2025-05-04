# The purpose of this file is to simply run make commands in deterministic
# environment for those interested in contributing. This is not to be used
# for production purposes.
FROM ubuntu:22.04

RUN apt update && apt upgrade

RUN apt install -y wget make vim git curl golang-go

# This is a workaround so that running `make lint` does not result in a 
# "error obtaining VCS status: exit status 128" error
RUN go env -w GOFLAGS="-buildvcs=false"