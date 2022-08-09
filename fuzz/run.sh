mkdir -p ./testdata/fuzz/FuzzMain
exit_status=0
while [ $exit_status -eq 0 ]; do
	go test -fuzz=Fuzz -fuzztime=15m -v >.stdout.log
	exit_status=$?
	if [ -n ""$(grep killed .stdout.log) ]; then
		echo "get killed, continue fuzzing"
		exit_status=0
	fi
done
rm .stdout.log
exit 1
