module github.com/cloudwego/frugal

go 1.16

require (
	github.com/chenzhuoyu/iasm v0.9.0
	github.com/davecgh/go-spew v1.1.1
	github.com/klauspost/cpuid/v2 v2.2.4
	github.com/oleiade/lane v1.0.1
	github.com/stretchr/testify v1.7.0
	golang.org/x/arch v0.2.0
	gonum.org/v1/gonum v0.12.0
)

// Frugal might be mistakenly using the object pool in iasm, which causes the panic 'lable was alreay linked'
// It's a fast workaround by removing the object pool in the replacement.
replace github.com/chenzhuoyu/iasm => github.com/felix021/iasm v0.9.1
