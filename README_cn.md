# Frugal

[English](README.md) | 中文

一种无需生成代码、高性能的动态 Thrift 编解码器。

## 特点

### 无需生成代码

传统的 Thrift 编解码方式，要求用户必须要先生成编解码代码，Frugal 通过反射 struct field tag 动态生成编解码器避免了这一过程。

### 高性能

基于 `frugal/tests` 的测试用例，Frugal 的性能 比 Apache Thrift (TBinaryProtocol) 好 1 到 4 倍。

不同的测试用例，结果可能会有些差异。欢迎给我们分享你的测试数据。

```text
go version go1.23.6 linux/amd64

goos: linux
goarch: amd64
pkg: github.com/cloudwego/frugal/tests
cpu: Intel(R) Xeon(R) Gold 5118 CPU @ 2.30GHz

Marshal_ApacheThrift/small-4             3468714         346.1 ns/op    1684.32 MB/s           0 B/op          0 allocs/op
Marshal_ApacheThrift/medium-4             128386          9343 ns/op    1875.07 MB/s           0 B/op          0 allocs/op
Marshal_ApacheThrift/large-4                7208        164521 ns/op    1845.68 MB/s         109 B/op          0 allocs/op
Marshal_Frugal/small-4                  13032746         92.45 ns/op    6306.09 MB/s           0 B/op          0 allocs/op
Marshal_Frugal/medium-4                   327564          3669 ns/op    4774.38 MB/s           0 B/op          0 allocs/op
Marshal_Frugal/large-4                     18751         64212 ns/op    4728.90 MB/s           0 B/op          0 allocs/op

Unmarshal_ApacheThrift/small-4           1548732         774.1 ns/op     753.15 MB/s        1120 B/op          4 allocs/op
Unmarshal_ApacheThrift/medium-4            42676         30665 ns/op     571.27 MB/s       44704 B/op        175 allocs/op
Unmarshal_ApacheThrift/large-4              2106        515642 ns/op     588.88 MB/s      775936 B/op       3030 allocs/op
Unmarshal_Frugal/small-4                 4963635         266.2 ns/op    2189.92 MB/s         544 B/op          1 allocs/op
Unmarshal_Frugal/medium-4                  99786         11321 ns/op    1547.45 MB/s       19908 B/op         57 allocs/op
Unmarshal_Frugal/large-4                    5838        197987 ns/op    1533.69 MB/s      349252 B/op        997 allocs/op
```

## 用 Frugal 可以做什么？

### 使用 Frugal 作为 [Kitex](https://github.com/cloudwego/kitex) 的编解码

可以不用再生成大量的编解码代码，使仓库变得干净整洁，review 时也不用再带上一堆无意义的 diff。然后相比于生成的编解码代码，Frugal 的性能更高。

### 在 [Thriftgo](https://github.com/cloudwego/thriftgo) 生成的 struct 上进行编解码

如果你只需要使用 Thrift 的编解码能力，同时也定义好了 IDL，那么只需要用 Thriftgo 生成 IDL 对应的 Go 语言 struct，就可以使用 Frugal 的编解码能力了。

### 直接定义 struct 进行编解码

如果你们连 IDL 都不想有，没问题，直接定义好 Go 语言 struct 后，给每个 Field 带上 Frugal 所需的 tag，就可以直接使用 Frugal 进行编解码了。

## 使用手册

### 配合 Kitex 使用

#### 1. 更新 Kitex 和 Frugal

```shell
go get github.com/cloudwego/kitex@latest
go get github.com/cloudwego/frugal@latest
```

#### 2. 带上 `-thrift frugal_tag` 参数重新生成一次代码

示例：

```shell
kitex -thrift frugal_tag -service a.b.c my.thrift
```

如果不需要编解码代码，可以带上 `-thrift template=slim` 参数，大幅减少生成的代码量。

```shell
kitex -thrift frugal_tag,template=slim -service a.b.c my.thrift
```

注：最新版本的 thriftgo（>= v0.3.0）默认生成的 `thrift` struct tag 已兼容 Frugal，因此对于不含 `list`、`set`、`enum` 字段的结构体，`frugal_tag` 参数是可选的。

#### 3. 初始化 client 和 server 时启用 Frugal 编解码

使用 `thrift.NewThriftCodecWithConfig` 启用 Frugal。编解码器优先级：Frugal > FastCodec > Apache Thrift。

client 示例：

```go
package main

import (
    "context"

    "example.com/kitex_test/client/kitex_gen/a/b/c/echo"
    "github.com/cloudwego/kitex/client"
    "github.com/cloudwego/kitex/pkg/remote/codec/thrift"
)

func main() {
    codec := thrift.NewThriftCodecWithConfig(thrift.FrugalRead | thrift.FrugalWrite)
    cli := echo.MustNewClient("a.b.c", client.WithPayloadCodec(codec))
    ...
}
```

server 示例：

```go
package main

import (
    "log"

    "github.com/cloudwego/kitex/server"
    c "example.com/kitex_test/kitex_gen/a/b/c/echo"
    "github.com/cloudwego/kitex/pkg/remote/codec/thrift"
)

func main() {
    codec := thrift.NewThriftCodecWithConfig(thrift.FrugalRead | thrift.FrugalWrite)
    svr := c.NewServer(new(EchoImpl), server.WithPayloadCodec(codec))

    err := svr.Run()
    if err != nil {
        log.Println(err.Error())
    }
}
```

注：对于没有 `frugal` struct tag 的类型，Kitex 会自动 fallback 到 FastCodec 或 Apache Thrift，因此可以放心全局启用 Frugal。

#### 编解码器选项

| 选项 | 说明 |
|------|------|
| `thrift.FrugalRead` | 使用 Frugal 进行反序列化 |
| `thrift.FrugalWrite` | 使用 Frugal 进行序列化 |
| `thrift.FrugalReadWrite` | `FrugalRead \| FrugalWrite` 的简写 |
| `thrift.FastRead` | 使用 FastCodec 进行反序列化 |
| `thrift.FastWrite` | 使用 FastCodec 进行序列化 |
| `thrift.EnableSkipDecoder` | 使用 Buffered 传输协议（无 Framed/TTHeader）时需要 |

使用 Framed 或 TTHeader 传输协议时，`FrugalRead | FrugalWrite` 即可。使用 Buffered（PurePayload）传输协议时，需额外添加 `EnableSkipDecoder`：

```go
codec := thrift.NewThriftCodecWithConfig(thrift.FrugalRead | thrift.FrugalWrite | thrift.EnableSkipDecoder)
```

### 配合 Thriftgo 做 Thrift IDL 的编解码

#### 编写 Thrift 文件

现在假设我们有如下 Thrift 文件：  
my.thrift:

```thrift
struct MyStruct {
    1: string msg
    2: i64 code
}
```

#### 使用 Thriftgo 生成代码

定义好需要的 Thrift 文件后，使用 Thriftgo 生成 Go 语言代码。

最新版本的 thriftgo（>= v0.3.0）默认生成的 `thrift` struct tag 已兼容 Frugal。对于包含 `list`、`set`、`enum` 字段的结构体，需添加 `frugal_tag` 参数生成显式的 Frugal tag：

```shell
thriftgo -r -o thrift -g go:frugal_tag,package_prefix=example.com/kitex_test/thrift my.thrift
```

如果不需要编解码代码，可以带上 `template=slim` 参数减少生成的代码量：

```shell
thriftgo -r -o thrift -g go:frugal_tag,template=slim,package_prefix=example.com/kitex_test/thrift my.thrift
```

#### 使用 Frugal 进行编解码

生成所需要的结构体后，直接使用 Frugal 进行编解码即可。  
示例：

```go
package main

import (
    "github.com/cloudwego/frugal"

    "example.com/kitex_test/thrift"
)

func main() {
    ms := &thrift.MyStruct{
        Msg: "my message",
        Code: 1024,
    }
    ...
    buf := make([]byte, frugal.EncodedSize(ms))
    frugal.EncodeObject(buf, nil, ms)
    ...
    got := &thrift.MyStruct{}
    frugal.DecodeObject(buf, got)
    ...
}
```

### 直接定义 struct 进行编解码

#### 定义 struct

现在假设我们需要如下 struct：

```go
type MyStruct struct {
    Msg     string
    Code    int64
    Numbers []int64 
}
```

#### 给结构体字段添加 tag

Frugal 所需的 tag 形如 `frugal:"1,default,string"`，其中 `1` 为字段 ID， `default` 为字段的 requiredness， `string` 表示字段的类型。字段 ID 和 字段 requiredness 是必须的，但是字段类型只有当字段为 `list` 、`set` 和 `enum` 时是必须的。

上述的 `MyStruct` 可以添加如下 tag：

```go
type MyStruct struct {
    Msg     string  `frugal:"1,default"`
    Code    int64   `frugal:"2,default"`
    Numbers []int64 `frugal:"3,default,list<i64>"`
}
```

下面是完整的类型示例：

```go
type MyEnum int64

type Example struct {
 MyOptBool         *bool            `frugal:"1,optional"`
 MyReqBool         bool             `frugal:"2,required"`
 MyOptByte         *int8            `frugal:"3,optional"`
 MyReqByte         int8             `frugal:"4,required"`
 MyOptI16          *int16           `frugal:"5,optional"`
 MyReqI16          int16            `frugal:"6,required"`
 MyOptI32          *int32           `frugal:"7,optional"`
 MyReqI32          int32            `frugal:"8,required"`
 MyOptI64          *int64           `frugal:"9,optional"`
 MyReqI64          int64            `frugal:"10,required"`
 MyOptString       *string          `frugal:"11,optional"`
 MyReqString       string           `frugal:"12,required"`
 MyOptBinary       []byte           `frugal:"13,optional"`
 MyReqBinary       []byte           `frugal:"14,required"`
 MyOptI64Set       []int64          `frugal:"15,optional,set<i64>"`
 MyReqI64Set       []int64          `frugal:"16,required,set<i64>"`
 MyOptI64List      []int64          `frugal:"17,optional,list<i64>"`
 MyReqI64List      []int64          `frugal:"18,required,list<i64>"`
 MyOptI64StringMap map[int64]string `frugal:"19,optional"`
 MyReqI64StringMap map[int64]string `frugal:"20,required"`
 MyOptEnum         *MyEnum          `frugal:"21,optional,i64"`
 MyReqEnum         *MyEnum          `frugal:"22,optional,i64"`
}
```

#### 使用 Frugal 进行编解码

直接使用 Frugal 进行编解码即可。  
示例：

```go
package main

import (
    "github.com/cloudwego/frugal"
)

func main() {
    ms := &thrift.MyStruct{
        Msg: "my message",
        Code: 1024,
        Numbers: []int64{0, 1, 2, 3, 4},
    }
    ...
    buf := make([]byte, frugal.EncodedSize(ms))
    frugal.EncodeObject(buf, nil, ms)
    ...
    got := &thrift.MyStruct{}
    frugal.DecodeObject(buf, got)
    ...
}
```
