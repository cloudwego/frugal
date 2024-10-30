module github.com/cloudwego/frugal/tests

go 1.18

require (
	github.com/apache/thrift v0.13.0
	github.com/cloudwego/frugal v0.2.0
	github.com/cloudwego/gopkg v0.1.2
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc
	github.com/google/gofuzz v1.0.0
	github.com/stretchr/testify v1.9.0
)

require (
	github.com/bytedance/gopkg v0.1.1 // indirect
	github.com/cloudwego/iasm v0.2.0 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	golang.org/x/arch v0.8.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace github.com/cloudwego/frugal => ../

replace github.com/apache/thrift => github.com/apache/thrift v0.13.0
