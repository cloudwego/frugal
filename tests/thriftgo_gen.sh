#! /bin/bash

# please install the latest thriftgo version for fastgo generator
# go install github.com/cloudwego/thriftgo@main
thriftgo -g fastgo:gen_setter=false,frugal_tag -o=. ./baseline.thrift
