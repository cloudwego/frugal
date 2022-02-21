package main

import (
    `fmt`

    `github.com/cloudwego/frugal`
    `github.com/cloudwego/frugal/testdata/kitex_gen/baseline`
)

var V int

var Obj []byte

func init() {
    obj := new(baseline.Simple)
    buf := make([]byte, frugal.EncodedSize(obj))
    _, err := frugal.EncodeObject(buf, nil, obj)
    if err != nil {
        panic(err)
    }
    Obj = buf
}

func F() { fmt.Printf("Hello, number %d\n", V) }

func Marshal(val interface{}) ([]byte, error) {
    buf := make([]byte, frugal.EncodedSize(val))
    _, err := frugal.EncodeObject(buf, nil, val)
    return buf, err
}
