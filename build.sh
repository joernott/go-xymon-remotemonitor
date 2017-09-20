#!/bin/bash
set -e
go get -v -d -u 
CGO_ENABLED=0 go build -a -o remotemonitor -ldflags "-w -s" main.go
VERSION=$(git describe --abbrev=0 --tags|sed -e 's/^v//')
tar -cJvf remotemonitor-${VERSION}.tar.xz remotemonitor
