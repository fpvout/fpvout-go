bin/livertp-static-linux-amd64: livertp-static.go
	env && GOOS=linux GOARCH=amd64 go build -o $@ $<
bin/livertp-static-windows-amd64: livertp-static.go
	CC=amd64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o $@ $<
bin/livertp-static-linux-mipsle: livertp.go
	CC=/usr/bin/mipsel-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -o $@ -v -ldflags "-linkmode external -extldflags -static" livertp-static.go
