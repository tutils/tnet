# tnet

插件化架构的网络开发工具包

## 特性

- **tcp** - TCP开发工具包。TCP服务器和客户端。
- **tun** - 数据隧道。任何可进行数据通信的逻辑，将在这里被抽象为Reader和Writer，默认管道通信协议为websocket。
- **proxy** - 代理。利用隧道将远端的TCP服务代理到本地。
- **crypt** - 加密。通过修饰实现Reader或Writer的加密。
- **cmd** - 命令解析。目前提供了两种子命令proxy和endpoint
- **tnet** - 命令行界面。

## 开发

基于插件化的架构设计，所以主要结构在创建的时候各种选项都是可定制的。

- **样例一** - 创建一个TCP服务器

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

type connIdKey struct{}

func (h *handler) ServeTCP(ctx context.Context, conn tcp.Conn) {
    // 函数结束自动关闭和清理当前TCP连接
	// 如果需要提供长连接服务，自行用循环结构进行控制
    connId := ctx.Value(connIdkey{}).(int64)
    conn.Writer().Write([]byte(fmt.Sprintf("I'm %d", connId)))
}

func main() {
    var connId int64 = 0

    h := &handler{}
    // 新建一个服务器
    srv := tcp.NewServer(
        // 监听地址
        tcp.WithListenAddress(":8080"),
    	// 所采用的TCP连接处理器，可以根据要提供的TCP服务的性质选择合适的处理器。这里使用原始TCP处理器
        tcp.WithServerHandler(tcp.NewRawTCPHandler(h)),
        // 新连接接入时候的上下文钩子函数，这里给每个连接分配一个ID
        tcp.WithServerConnContextFunc(func(ctx context.Context, c net.Conn) context.Context {
            connId++
            return context.WithValue(ctx, connIdKey{}, connId)
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

- **样例二** - 创建一个本地代理

```go
package main

import (
    "github.com/tutils/tnet/crypt/xor"
    "github.com/tutils/tnet/proxy"
    "github.com/tutils/tnet/tun"
    "log"
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
                // 隧道连接处理器
                tun.WithClientHandler(proxy.NewTunClientHandler()),
            ),
        ),
        // 本地代理监听的地址
        proxy.WithListenAddress(":1022"),
        // 远端代理访问的地址
        proxy.WithConnectAddress("127.0.0.1:22"),
        // 代理所用的数据隧道将会被加密
        proxy.WithTunClientCrypt(xor.NewCrypt(1234)),
    )
    // 启动代理
    // 当然如果需要提供完整TCP代理服务，还需要在远端启动一个endpoint
    if err := p.DialAndServe(); err != nil {
        log.Fatalln(err)
    }
}
```

## 扩展

tnet通过接口化的设计以及创建对象时的选项化配置实现的组件插件化特性。
如果需要需要替换tnet中的某一个组件，可以对接口进行自行实现，并在创建持有该组件的对象时通过选项进行设置。
例如可以通过实现tnet/tun中的Server和Client接口，实现对proxy/endpoint中数据隧道的替换；通过实现tnet/crypt中的Crypt接口以支持不同的加密方案；等等。

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
    proxy.WithTunClientCrypt(aes.NewCrypt("akd8f=3ng0$5a@e9")),
    ...
)

```
## 命令行界面

详见 ```tnet --help```

## 协议

tnet已获得Apache 2.0许可。
