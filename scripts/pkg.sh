export GOARCH=amd64 && export GOOS=linux && go build --ldflags="-w -s" -o "a10000" ./
upx -9 a10000
echo "finished"