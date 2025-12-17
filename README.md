# napcat-go

- ✅ **强类型事件**：`PrivateMessageEvent`、`GroupMessageEvent` 等，字段清晰无冗余
- ✅ **多处理器支持**：可注册多个独立 handler，互不干扰  
- ✅ **中间件支持**：全局/私聊/群聊 middleware，可拦截/短路/包裹 handler  
- ✅ **安全执行**：单个 handler panic 不影响其他（可通过 `OnPanic` 捕获）  
- ✅ **低抽象开销**：事件解析基于 `gjson`，不走 `encoding/json` 反射解析
- ✅ **原生 NapCat 兼容**：自动过滤 `heartbeat/lifecycle`，解析 OneBot v11 消息段

> 适用于个人机器人、自动化工具、群管插件等场景。

---

## 📦 安装

```bash
go get github.com/ckikoo/napcat_ws
```

---

## 🚀 快速开始（连接 NapCat WS）

```go
package main

import (
	"context"
	"errors"
	"os/signal"
	"syscall"

	napcat "github.com/ckikoo/napcat_ws"
	"go.uber.org/zap"
)

func main() {
	bot := napcat.New("ws://127.0.0.1:3001/?access_token=YOUR_TOKEN")

	logger, _ := zap.NewDevelopment()
	defer func() { _ = logger.Sync() }()

	bot.OnError(func(err error) { logger.Error("bot error", zap.Error(err)) })
	bot.OnPanic(func(p napcat.PanicInfo) { logger.Error("panic", zap.Any("recovered", p.Recovered), zap.ByteString("stack", p.Stack)) })

	bot.OnPrivate(func(e *napcat.PrivateMessageEvent) {
		logger.Info("private message", zap.String("nick", e.Sender.Nickname), zap.Int64("user_id", e.UserID), zap.String("text", e.GetText()))
	})

	bot.OnGroup(func(e *napcat.GroupMessageEvent) {
		logger.Info("group message", zap.Int64("group_id", e.GroupID), zap.String("nick", e.Sender.Nickname), zap.Int64("user_id", e.UserID), zap.String("text", e.GetText()))
	})

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := bot.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Fatal("bot exit", zap.Error(err))
	}
}
```

运行示例：

```bash
go run ./cmd/ws-example -url "ws://127.0.0.1:3001/?access_token=YOUR_TOKEN"
```

---

## ✉️ 发送 Action 示例

```go
payload, _ := napcat.MarshalGroupTextMsg(123456, "hello")
_ = bot.Send(context.Background(), payload)
```

---

## 🧪 纯解析示例（不连接 WS）

```bash
go run ./main
```

---

## ✅ 开发检查

```bash
go test ./... -count=1
go vet ./...
```

### golangci-lint

安装：

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.8
```

运行：

```bash
golangci-lint run ./...
```
