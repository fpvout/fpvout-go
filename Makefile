EXECUTABLE=bin/livertp
VERSION=$(shell git rev-parse --abbrev-ref HEAD)-$(shell git describe --tags --always --long --dirty)
HOSTNAME=$(shell hostname)
TIMESTAMP=$(shell date)
USER=$(shell id -u -n)
LDFLAGS=-ldflags='-s -w -X main.COMPILE_VERSION=$(VERSION) -X main.COMPILE_HOSTNAME=$(HOSTNAME) -X "main.COMPILE_TIMESTAMP=$(TIMESTAMP)" -X "main.COMPILE_USER=$(USER)"'
INPUT_FILES=livertp-static.go

LINUX_AMD64=$(EXECUTABLE)_linux_amd64-$(VERSION)
LINUX_ARM64=$(EXECUTABLE)_linux_arm64-$(VERSION)
LINUX_ARMV7=$(EXECUTABLE)_linux_armv7-$(VERSION)
LINUX_ARMV6=$(EXECUTABLE)_linux_armv6-$(VERSION)
LINUX_ARMV5=$(EXECUTABLE)_linux_armv5-$(VERSION)
LINUX_MIPSEL=$(EXECUTABLE)_linux_mipsel-$(VERSION)
WINDOWS_AMD64=$(EXECUTABLE)_windows_amd64-$(VERSION).exe

all: clean linux windows
.PHONY: all clean upload deps ci

clean:
	rm bin/* 2>/dev/null; true
upload:
	mc cp bin/* minio/private/go-fpv/$(VERSION)/
	mc share download --expire=72h minio/private/go-fpv/$(VERSION)/
deps:
	go get ./...
	sudo apt-get update && sudo apt-get install -y gcc-aarch64-linux-gnu gcc-mingw-w64-x86-64 gcc-mipsel-linux-gnu gcc-arm-linux-gnueabihf gcc-arm-linux-gnueabi
ci: clean deps

linux: $(LINUX_ARM64) $(LINUX_ARMV7) $(LINUX_ARMV6 )$(LINUX_ARMV5) $(LINUX_AMD64) $(LINUX_MIPSEL)
windows: $(WINDOWS_AMD64)

$(LINUX_AMD64): $(INPUT_FILES)
	CC=x86_64-linux-gnu-gcc GOARCH=amd64 GOOS=linux go build -o $@ $(LDFLAGS) $<
$(LINUX_ARM64): $(INPUT_FILES)
	CC=aarch64-linux-gnu-gcc GOARCH=arm64 GOOS=linux CGO_ENABLED=1 go build -o $@ $(LDFLAGS) $<
$(LINUX_ARMV7): $(INPUT_FILES)
	CC=arm-linux-gnueabi-gcc GOARCH=arm GOARM=7 GOOS=linux CGO_ENABLED=1 go build -o $@ $(LDFLAGS) $<
$(LINUX_ARMV6): $(INPUT_FILES)
	CC=arm-linux-gnueabi-gcc GOARCH=arm GOARM=6 GOOS=linux CGO_ENABLED=1 go build -o $@ $(LDFLAGS) $<
$(LINUX_ARMV5): $(INPUT_FILES)
	CC=arm-linux-gnueabi-gcc GOARCH=arm GOARM=5 GOOS=linux CGO_ENABLED=1 go build -o $@ $(LDFLAGS) $<
$(LINUX_MIPSEL): $(INPUT_FILES)
	CC=mipsel-linux-gnu-gcc GOARCH=mipsle GOMIPS=softfloat GOOS=linux CGO_ENABLED=1 go build -o $@ $(LDFLAGS) $<
$(WINDOWS_AMD64): $(INPUT_FILES)
	CC=x86_64-w64-mingw32-gcc GOARCH=amd64 GOOS=windows CGO_ENABLED=1 go build -o $@ $(LDFLAGS) $<