#!/bin/bash
export GOMAXPROCS=4
go version
go test -run=^$ -bench="BenchmarkAllSize_(Marshal|Unmarshal)_(ApacheThrift|Frugal)" -benchmem
