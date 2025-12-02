# TNet

[![Windows](https://img.shields.io/badge/-Windows-blue?logo=windows)](https://github.com/tutils/tnet/releases)
[![MacOS](https://img.shields.io/badge/-MacOS-black?logo=apple)](https://github.com/tutils/tnet/releases)
[![Linux](https://img.shields.io/badge/-Linux-purple?logo=ubuntu)](https://github.com/tutils/tnet/releases)
  
[![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Build Status](https://github.com/tutils/tnet/actions/workflows/build.yml/badge.svg)](https://github.com/tutils/tnet/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tutils/tnet)](https://goreportcard.com/report/github.com/tutils/tnet)

[English](README.md)

插件化架构的网络开发工具包。

## 特性

- **tcp** - TCP开发工具包。TCP服务器和客户端。
- **tun** - 数据隧道。任何可进行数据通信的逻辑，将在这里被抽象为Reader和Writer，默认管道通信协议为websocket。
- **endpoint** - 端。端到端通过隧道通信。默认的隧道处理器可将远端的TCP服务代理到本地。
- **crypt** - 加密。通过修饰实现Reader或Writer的加密。
- **cmd** - 命令解析。目前提供了四种子命令：proxy、agent、server和httpsrv。
- **tnet** - 命令行界面。

## 开发

基于插件化的架构设计，所以主要结构在创建的时候各种选项都是可定制的。

- **样例一** - 创建一个TCP服务器：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net"

    "github.com/tutils/tnet/tcp"
)

type handler struct {
}

type connIDKey struct{}

func (h *handler) ServeTCP(ctx context.Context, conn tcp.Conn) {
    // 函数结束自动关闭和清理当前TCP连接
    // 如果需要提供长连接服务，自行用循环结构进行控制
    connID := ctx.Value(connIDkey{}).(int64)
    conn.Writer().Write([]byte(fmt.Sprintf("I'm %d", tunID, connID)))
}

func main() {
    var connID int64 = 0

    h := &handler{}
    // 新建一个服务器
    srv := tcp.NewServer(
        // 监听地址
        tcp.WithListenAddress(":8080"),
        // 所采用的TCP连接处理器，可以根据要提供的TCP服务的性质选择合适的处理器。这里使用原始TCP处理器
        tcp.WithServerHandler(tcp.NewRawTCPHandler(h)),
        // 新连接接入时候的上下文钩子函数，这里给每个连接分配一个ID
        tcp.WithServerConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
            connID++
            return context.WithValue(ctx, connIDKey{}, connID)
        }),
    )
    // 结束时优雅的关闭服务器
    // 直到全部连接不在活跃时才会完成关闭操作，如果需要强行关闭可调用srv.Close()
    defer srv.Shutdown(context.Background())
    // 启动服务
    if err := srv.ListenAndServe(); err != nil {
        log.Fatalln(err)
    }
}
```

- **样例二** - 创建一个本地代理：

```go
package main

import (
    "context"
    "log"

    "github.com/tutils/tnet/crypt/xor"
    "github.com/tutils/tnet/proxy"
    "github.com/tutils/tnet/tun"
)

func main() {
    // 新建一个代理
    p := proxy.NewProxy(
        // 所采用的隧道客户端
        proxy.WithTunClient(
            // 新建一个默认隧道客户端
            tun.NewClient(
                // 隧道连接地址
                tun.WithConnectAddress("ws://127.0.0.1:8080/stream"),
            ),
        ),
        // 隧道连接处理器
        tun.WithTunHandlerNewer(proxy.NewTCPProxyTunHandler),
        // 本地代理监听的地址
        proxy.WithListenAddress(":1022"),
        // 远端代理访问的地址
        proxy.WithConnectAddress("127.0.0.1:22"),
        // 代理所用的数据隧道将会被加密
        proxy.WithTunCrypt(xor.NewCrypt(1234)),
    )
    // 启动代理
    // 当然如果需要提供完整TCP代理服务，还需要在远端启动一个agent
    if err := p.Serve(context.Background()); err != nil {
        log.Fatalln(err)
    }
}
```

## 扩展

tnet通过接口化的设计以及创建对象时的选项化配置实现的组件插件化特性。
如果需要需要替换tnet中的某一个组件，可以对接口进行自行实现，并在创建持有该组件的对象时通过选项进行设置。
例如可以通过实现tnet/tun中的Server和Client接口，实现对proxy/agent中数据隧道的替换；通过实现tnet/crypt中的Crypt接口以支持不同的加密方案；等等。

```go
p := proxy.NewProxy(
    proxy.WithTunClient(
        // 将隧道替换成redis方案(比如可通过pub/sub进行数据通信)
        redis.NewClient(
            // redis隧道的创建选项
            redis.WithConfig(...),
            ...
        ),
        ...
    ),
    // 对解密协议进行替换
    proxy.WithTunCrypt(aes.NewCrypt("akd8f=3ng0$5a@e9")),
    ...
)
```

## 命令行界面

### 命令概览

```shell
tnet [flags]
tnet [command]
```

### 可用命令

- **agent** - TCP隧道代理服务端
- **proxy** - TCP隧道代理客户端
- **server** - 启动tnet管理服务器
- **httpsrv** - HTTP文件服务器
- **completion** - 为您的shell生成自动补全脚本

### 命令用法

#### 1. Proxy 命令

启动TCP隧道代理：

```bash
# 正常模式：proxy主动连接到agent
tnet proxy --listen=0.0.0.0:56080 --connect=127.0.0.1:3128 --tunnel-connect=ws://123.45.67.89:8080/stream --crypt-key=816559

# 反向模式：proxy等待agent连接
tnet proxy --tunnel-listen=ws://0.0.0.0:8080/stream --connect=127.0.0.1:3128 --crypt-key=816559

# 带命令执行功能
tnet proxy --listen=0.0.0.0:56080 --execute="ls -la" --tunnel-connect=ws://123.45.67.89:8080/stream --crypt-key=816559
```

#### 2. Agent 命令

启动TCP隧道代理客户端：

```bash
# 正常模式：agent等待proxy连接
tnet agent --tunnel-listen=ws://0.0.0.0:8080/stream --crypt-key=816559

# 反向模式：agent主动连接到proxy
tnet agent --tunnel-connect=ws://proxy-server:8080/stream --crypt-key=816559

# 启用远程命令执行（安全警告：仅在可信输入时使用）
tnet agent --tunnel-listen=ws://0.0.0.0:8080/stream --enabled-execute --crypt-key=816559
```

#### 3. Server 命令

启动带web界面的tnet管理服务器：

```bash
tnet server --listen=0.0.0.0:8080
```

#### 4. HTTPSrv 命令

启动HTTP文件服务器，支持文件浏览、上传和下载功能：

```bash
tnet httpsrv --listen=0.0.0.0:8080
```

#### 5. Completion 命令

为您的shell生成自动补全脚本：

```bash
# Bash
eval "$(tnet completion bash)"

# Zsh
eval "$(tnet completion zsh)"

# 添加到shell配置文件中以永久生效
echo "eval \"$(tnet completion $(basename $SHELL))\"" >> ~/.bashrc
```

## 将TNet作为系统服务使用

### Linux (Systemd)

1. 复制服务文件：

```bash
sudo cp /path/to/tnet/service/tnet.service /etc/systemd/system/
```

2. 编辑服务文件以匹配您的配置：

```bash
sudo vi /etc/systemd/system/tnet.service
```

3. 启用并启动服务：

```bash
sudo systemctl enable tnet.service
sudo systemctl start tnet.service
sudo systemctl status tnet.service
```

### macOS (Launchd)

1. 将plist文件复制到LaunchAgents目录：

```bash
sudo cp /path/to/tnet/service/com.tutils.tnet.proxy.plist /Library/LaunchAgents/
```

2. 编辑plist文件以匹配您的配置：

```bash
sudo vi /Library/LaunchAgents/com.tutils.tnet.proxy.plist
```

3. 加载并启动服务：

```bash
sudo launchctl load /Library/LaunchAgents/com.tutils.tnet.proxy.plist
sudo launchctl start com.tutils.tnet.proxy
```

## 服务配置文件

- **tnet.service** - Linux系统的Systemd服务文件
- **com.tutils.tnet.proxy.plist** - macOS系统的Launchd plist文件
- **completion.sh** - Bash和Zsh shell的自动补全脚本

## 协议

tnet已获得Apache 2.0许可。 