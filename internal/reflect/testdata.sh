thriftgo -r -o tmp -g go:scan_value_for_enum=false,gen_setter=false,template=slim,package_prefix=. testdata.thrift

# Add copyright header to the generated file
cat > testdata_test.go << 'EOF'
/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

EOF
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
  sed '/DEFAULT/d' >> testdata_test.go

gofmt -w ./testdata_test.go
rm -rf ./tmp/
