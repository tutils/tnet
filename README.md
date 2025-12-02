# TNet

[![Windows](https://img.shields.io/badge/-Windows-blue?logo=windows)](https://github.com/tutils/tnet/releases)
[![MacOS](https://img.shields.io/badge/-MacOS-black?logo=apple)](https://github.com/tutils/tnet/releases)
[![Linux](https://img.shields.io/badge/-Linux-purple?logo=ubuntu)](https://github.com/tutils/tnet/releases)
  
[![License](https://img.shields.io/:license-apache-blue.svg)](https://opensource.org/licenses/Apache-2.0)
[![Build Status](https://github.com/tutils/tnet/actions/workflows/build.yml/badge.svg)](https://github.com/tutils/tnet/actions/workflows/build.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/tutils/tnet)](https://goreportcard.com/report/github.com/tutils/tnet)

[中文](README_zh.md)

A plugin-based network development toolkit.

## Features

- **tcp** - TCP development toolkit. TCP server and client.
- **tun** - Data tunnel. Any logic that can perform data communication will be abstracted here as Reader and Writer, with websocket as the default pipeline communication protocol.
- **endpoint** - Endpoint. End-to-end communication through tunnels. The default tunnel handler can proxy remote TCP services to local.
- **crypt** - Encryption. Implement encryption by decorating Reader or Writer.
- **cmd** - Command parsing. Currently provides four subcommands: proxy, agent, server, and httpsrv.
- **tnet** - Command line interface.

## Development

Based on a plugin-based architecture design, the main structures are customizable with various options when created.

- **Example 1** - Create a TCP server:

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
    // The function automatically closes and cleans up the current TCP connection when it ends
    // If you need to provide long connection service, use loop structure for control
    connID := ctx.Value(connIDkey{}).(int64)
    conn.Writer().Write([]byte(fmt.Sprintf("I'm %d", tunID, connID)))
}

func main() {
    var connID int64 = 0

    h := &handler{}
    // Create a new server
    srv := tcp.NewServer(
        // Listen address
        tcp.WithListenAddress(":8080"),
        // TCP connection handler to be used, choose appropriate handler based on the nature of TCP service to be provided. Here we use raw TCP handler
        tcp.WithServerHandler(tcp.NewRawTCPHandler(h)),
        // Context hook function for new connection access, here assigning an ID to each connection
        tcp.WithServerConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
            connID++
            return context.WithValue(ctx, connIDKey{}, connID)
        }),
    )
    // Gracefully close the server when finished
    // Will not complete shutdown until all connections are no longer active, call srv.Close() if forced shutdown is needed
    defer srv.Shutdown(context.Background())
    // Start service
    if err := srv.ListenAndServe(); err != nil {
        log.Fatalln(err)
    }
}
```

- **Example 2** - Create a local proxy:

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
    // Create a new proxy
    p := proxy.NewProxy(
        // Tunnel client to be used
        proxy.WithTunClient(
            // Create a new default tunnel client
            tun.NewClient(
                // Tunnel connection address
                tun.WithConnectAddress("ws://127.0.0.1:8080/stream"),
            ),
        ),
        // Tunnel connection handler
        tun.WithTunHandlerNewer(proxy.NewTCPProxyTunHandler),
        // Local proxy listen address
        proxy.WithListenAddress(":1022"),
        // Remote proxy access address
        proxy.WithConnectAddress("127.0.0.1:22"),
        // The data tunnel used by the proxy will be encrypted
        proxy.WithTunCrypt(xor.NewCrypt(1234)),
    )
    // Start proxy
    // Of course, to provide complete TCP proxy service, you also need to start an agent on the remote end
    if err := p.Serve(context.Background()); err != nil {
        log.Fatalln(err)
    }
}
```

## Extension

tnet implements component plugin features through interface-based design and option-based configuration when creating objects.
If you need to replace a component in tnet, you can implement the interface yourself and set it through options when creating objects that hold that component.
For example, you can implement the Server and Client interfaces in tnet/tun to replace the data tunnel in proxy/agent; implement the Crypt interface in tnet/crypt to support different encryption schemes; and so on.

```go
p := proxy.NewProxy(
    proxy.WithTunClient(
        // Replace tunnel with redis solution (e.g., data communication through pub/sub)
        redis.NewClient(
            // Creation options for redis tunnel
            redis.WithConfig(...),
            ...
        ),
        ...
    ),
    // Replace decryption protocol
    proxy.WithTunCrypt(aes.NewCrypt("akd8f=3ng0$5a@e9")),
    ...
)
```

## Command Line Interface

### Command Overview

```shell
tnet [flags]
tnet [command]
```

### Available Commands

- **agent** - TCP tunnel agent
- **proxy** - TCP tunnel proxy
- **server** - Start tnet management server
- **httpsrv** - HTTP file server
- **completion** - Generate completion script for your shell

### Command Usage

#### 1. Proxy Command

Start TCP tunnel proxy:

```bash
# Normal mode: proxy actively connects to agent
tnet proxy --listen=0.0.0.0:56080 --connect=127.0.0.1:3128 --tunnel-connect=ws://123.45.67.89:8080/stream --crypt-key=816559

# Reverse mode: proxy waits for agent to connect
tnet proxy --tunnel-listen=ws://0.0.0.0:8080/stream --connect=127.0.0.1:3128 --crypt-key=816559

# With command execution
tnet proxy --listen=0.0.0.0:56080 --execute="ls -la" --tunnel-connect=ws://123.45.67.89:8080/stream --crypt-key=816559
```

#### 2. Agent Command

Start TCP tunnel agent:

```bash
# Normal mode: agent waits for proxy to connect
tnet agent --tunnel-listen=ws://0.0.0.0:8080/stream --crypt-key=816559

# Reverse mode: agent actively connects to proxy
tnet agent --tunnel-connect=ws://proxy-server:8080/stream --crypt-key=816559

# Enable remote command execution (SECURITY WARNING: only use with trusted input)
tnet agent --tunnel-listen=ws://0.0.0.0:8080/stream --enabled-execute --crypt-key=816559
```

#### 3. Server Command

Start tnet management server with web interface:

```bash
tnet server --listen=0.0.0.0:8080
```

#### 4. HTTPSrv Command

Start HTTP file server with file browsing, uploading, and downloading capabilities:

```bash
tnet httpsrv --listen=0.0.0.0:8080
```

#### 5. Completion Command

Generate completion script for your shell:

```bash
# Bash
eval "$(tnet completion bash)"

# Zsh
eval "$(tnet completion zsh)"

# Add to your shell config file for permanent effect
echo "eval \"$(tnet completion $(basename $SHELL))\"" >> ~/.bashrc
```

## Using TNet as a System Service

### Linux (Systemd)

1. Copy the service file:

```bash
sudo cp /path/to/tnet/service/tnet.service /etc/systemd/system/
```

2. Edit the service file to match your configuration:

```bash
sudo vi /etc/systemd/system/tnet.service
```

3. Enable and start the service:

```bash
sudo systemctl enable tnet.service
sudo systemctl start tnet.service
sudo systemctl status tnet.service
```

### macOS (Launchd)

1. Copy the plist file to LaunchAgents directory:

```bash
sudo cp /path/to/tnet/service/com.tutils.tnet.proxy.plist /Library/LaunchAgents/
```

2. Edit the plist file to match your configuration:

```bash
sudo vi /Library/LaunchAgents/com.tutils.tnet.proxy.plist
```

3. Load and start the service:

```bash
sudo launchctl load /Library/LaunchAgents/com.tutils.tnet.proxy.plist
sudo launchctl start com.tutils.tnet.proxy
```

## Service Configuration Files

- **tnet.service** - Systemd service file for Linux
- **com.tutils.tnet.proxy.plist** - Launchd plist file for macOS
- **completion.sh** - Auto-completion script for bash and zsh shells

## License

tnet is licensed under the Apache License 2.0.
