#!/bin/bash

#set -x

TARGET="tnet"

export CGO_ENABLED="0"

go mod tidy

LDFLAGS="-s -w"
TARGET="tnet"  # 请替换为你的目标程序名

# 定义架构和操作系统组合，使用 os/arch 的形式
COMBINATIONS=(
    "linux/amd64"
    "darwin/amd64"
    "windows/amd64"
    "linux/arm64"
    "darwin/arm64"
    "windows/arm64"
)

# 遍历组合进行编译和打包
for COMBO in "${COMBINATIONS[@]}"; do
    # 分割组合字符串为 GOOS 和 GOARCH
    IFS="/" read -r GOOS GOARCH <<< "$COMBO"

    # 为 Windows 平台指定可执行文件的后缀
    if [ "$GOOS" == "windows" ]; then
        EXT=".exe"
    else
        EXT=""
    fi

    # 编译并打包
    GOARCH=$GOARCH GOOS=$GOOS go build -ldflags="$LDFLAGS" -o $TARGET$EXT && \
    (zip -r -q -o $TARGET-$GOOS-$GOARCH.zip $TARGET$EXT; \
    rm $TARGET$EXT)
done