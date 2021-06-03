EXECUTABLE=bin/livertp
VERSION=$(shell git rev-parse --abbrev-ref HEAD)-$(shell git describe --tags --always --long --dirty)
HOSTNAME=$(shell hostname)
TIMESTAMP=$(shell date)
USER=$(shell id -u -n)
LDFLAGS=-ldflags='-s -w -X main.COMPILE_VERSION=$(VERSION) -X main.COMPILE_HOSTNAME=$(HOSTNAME) -X "main.COMPILE_TIMESTAMP=$(TIMESTAMP)" -X "main.COMPILE_USER=$(USER)"'
INPUT_FILES=livertp-static.go

LINUX_AMD64=$(EXECUTABLE)_linux_amd64-$(VERSION)
LINUX_ARM64=$(EXECUTABLE)_linux_arm64-$(VERSION)
DARWIN_AMD64=$(EXECUTABLE)_darwin_amd64-$(VERSION)
FREEBSD_AMD64=$(EXECUTABLE)_freebsd_amd64-$(VERSION)
WINDOWS_AMD64=$(EXECUTABLE)_windows_amd64-$(VERSION).exe

all: clean linux darwin freebsd windows
.PHONY: all clean upload

clean:
	rm bin/*; true
upload:
	mc cp bin/* minio/private/go-fpv/
deps:
	go get github.com/karalabe/gousb/usb
	sudo apt-get install gcc-aarch64-linux-gnu

linux: $(LINUX_ARM64) $(LINUX_AMD64)
darwin: $(DARWIN_AMD64)
freebsd: $(FREEBSD_AMD64) 
windows: $(WINDOWS_AMD64)

$(LINUX_ARM64): $(INPUT_FILES)
	CC=aarch64-linux-gnu-gcc GOARCH=arm64 GOOS=linux CGO_ENABLED=1 go build -o $@ $(LDFLAGS) $<
$(LINUX_AMD64): $(INPUT_FILES)
	GOOS=linux GOARCH=amd64 go build -o $@ $(LDFLAGS) $<
bin/livertp-static-windows-amd64: livertp-static.go
	CC=amd64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build $(LDFLAGS) -o $@ $<
bin/livertp-static-linux-mipsle: livertp-static.go
	CC=/usr/bin/mipsel-linux-gnu-gcc CGO_ENABLED=1 GOOS=linux GOARCH=mipsle GOMIPS=softfloat go build -o $@ -v -ldflags "-linkmode external -extldflags -static" $<
