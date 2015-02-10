#!/usr/bin/env sh

cd /go/src/github.com/rif/spark/
rm -rf build; mkdir build

go build -v -o build/linux_amd64
GOOS=linux GOARCH=386 go build -v -o build/spark_linux_386
GOOS=linux GOARCH=arm GOARM=5 go build -v -o build/spark_linux_arm5
GOOS=darwin GOARCH=amd64 go build -v -o build/spark_darwin_amd64
GOOS=darwin GOARCH=386 go build -v -o build/spark_darwin_386
GOOS=windows GOARCH=386 go build -v -o build/spark_windows_386.exe
GOOS=windows GOARCH=amd64 go build -v -o build/spark_windows_amd64.exe

go get github.com/pwaller/goupx/
/go/bin/goupx build/linux_amd64
upx build/*
chmod -R a+rw build
