module github.com/cloudwego/frugal/testdata

go 1.17

require (
	github.com/apache/thrift v0.12.0
	github.com/cloudwego/frugal v0.0.0-00010101000000-000000000000
	github.com/davecgh/go-spew v1.1.1
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200313102051-9f266ea9e77c // indirect
)

replace github.com/cloudwego/frugal => ../
