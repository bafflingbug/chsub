# chsub - 一个简单的Go语言发布订阅库

[English](./README.md)/中文

## 简介
`chsub` 是一个轻量级的发布订阅（Pub/Sub）库，专为Go语言设计，用于在协程之间进行简单的消息广播。

## 安装
```bash
go get github.com/bafflingbug/chsub@latest
```

## 使用
### 基础示例
```go
package main

import (
	"fmt"
	"log"

	"github.com/bafflingbug/chsub"
)

func main() {
	sub := chsub.NewSub()

	publish, _ := sub.NewTopic("topic_name", chsub.WithLength(10))

	subCh := make(chan any, 10)
	_ = sub.Sub(
		"topic_name", subCh, chsub.WithSubOnDrop(
			func(a any) {
				log.Printf("drop msg: %+v", a)
			},
		),
	)

	go func() {
		for msg := range subCh {
			fmt.Println("Received:", msg)
		}
	}()

	_ = publish("Hello, World!")
}
```

### 配置选项
#### 主题选项
- `WithLength(maxLength int)`: 设置主题缓冲区的最大长度。

#### 订阅者选项
- `WithSubOnDrop(f func(any))`: 设置消息丢弃时的回调函数。
