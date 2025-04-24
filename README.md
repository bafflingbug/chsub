# chsub - A Simple Pub/Sub Library for Go

English/[中文](./README_CN.md)

## Introduction
`chsub` is a lightweight Pub/Sub (Publish-Subscribe) library for Go, designed for simple message broadcasting between goroutines.

## Installation
```bash
go get github.com/bafflingbug/chsub@latest
```

## Usage
### Basic Example

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

### Configuration Options
#### Topic Options
- `WithLength(maxLength int)`: Sets the maximum length of the topic buffer.

#### Subscriber Options
- `WithSubOnDrop(f func(any))`: Sets a callback for dropped messages.
