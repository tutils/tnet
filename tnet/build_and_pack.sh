#!/bin/sh

#set -x

TARGET="tnet"

export CGO_ENABLED="0"

go mod tidy

export GOARCH="amd64"

export GOOS="linux"
go build && \
    (zip -r -q -o $TARGET-$GOOS-$GOARCH.zip $TARGET; \
    rm $TARGET)

export GOOS="darwin"
go build && \
    (zip -r -q -o $TARGET-$GOOS-$GOARCH.zip $TARGET; \
    rm $TARGET)

export GOOS="windows"
go build && \
    (zip -r -q -o $TARGET-$GOOS-$GOARCH.zip $TARGET.exe; \
    rm $TARGET.exe)

export GOARCH="arm64"

export GOOS="linux"
go build && \
    (zip -r -q -o $TARGET-$GOOS-$GOARCH.zip $TARGET; \
    rm $TARGET)

export GOOS="darwin"
go build && \
    (zip -r -q -o $TARGET-$GOOS-$GOARCH.zip $TARGET; \
    rm $TARGET)

export GOOS="windows"
go build && \
    (zip -r -q -o $TARGET-$GOOS-$GOARCH.zip $TARGET.exe; \
    rm $TARGET.exe)
