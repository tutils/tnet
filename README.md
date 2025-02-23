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
- **cmd** - Command parsing. Currently provides two subcommands: proxy and agent.
- **tnet** - Command line interface.

## Development

Based on a plugin-based architecture design, the main structures are customizable with various options when created.

- **Example 1** - Create a TCP server:

```go
package main

import (
    "context"
    "fmt"
    "github.com/tutils/tnet/tcp"
    "log"
    "net"
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
    "github.com/tutils/tnet/crypt/xor"
    "github.com/tutils/tnet/proxy"
    "github.com/tutils/tnet/tun"
    "log"
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
    if err := p.Serve(); err != nil {
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

See ```tnet --help``` for details.

## License

tnet is licensed under the Apache License 2.0.
