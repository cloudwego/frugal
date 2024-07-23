thriftgo -r -o tmp -g go:frugal_tag,scan_value_for_enum=false,gen_setter=false,template=slim,package_prefix=. testdata.thrift

# rm unused funcs and vars, keep the file smaller:
# func GetXXX
# func IsSet
# func String
# multiline DEFAULT vars
# singleline DEFAULT vars
sed '/func.* Get.* {/,/^}/d' ./tmp/reflect/testdata.go |\
  sed '/func.* IsSet.* {/,/^}/d' |\
  sed '/func.* String.* {/,/^}/d' |\
  sed '/DEFAULT.*{/,/^}/d' |\
  sed '/DEFAULT/d' > testdata_test.go

gofmt -w ./testdata_test.go
rm -rf ./tmp/
