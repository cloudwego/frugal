mkdir -p ./testdata/fuzz/FuzzMain
exit_status=0
while [ $exit_status -eq 0 ]; do
	go test -fuzz=Fuzz -fuzztime=15m -v
	exit_status=$?
done