# bot-ctl 构建（Linux 专用产物）
BIN      := bot-ctl
PKG      := ./cmd/bot-ctl
VERSION  ?= $(shell git describe --tags --always 2>/dev/null || echo dev)
COMMIT   ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE     := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS  := -s -w \
	-X github.com/bot-ctl/bot-ctl/internal/version.Version=$(VERSION) \
	-X github.com/bot-ctl/bot-ctl/internal/version.GitCommit=$(COMMIT) \
	-X github.com/bot-ctl/bot-ctl/internal/version.BuildDate=$(DATE)

.PHONY: all tidy build build-linux test vet clean bridge-image dist

all: build

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./...

build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o bin/$(BIN) $(PKG)

# 交叉编译 Linux amd64 + arm64
build-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/$(BIN)_linux_amd64 $(PKG)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags="$(LDFLAGS)" -o dist/$(BIN)_linux_arm64 $(PKG)

# 打包发布产物（含校验和）
dist: build-linux
	cd dist && \
	for a in amd64 arm64; do \
	  cp $(BIN)_linux_$$a $(BIN) && \
	  tar -czf $(BIN)_linux_$$a.tar.gz $(BIN) && \
	  rm -f $(BIN) && \
	  sha256sum $(BIN)_linux_$$a.tar.gz > $(BIN)_linux_$$a.tar.gz.sha256; \
	done

bridge-image:
	docker build -f build/bridge.Dockerfile -t botctl/qq-resource-bridge:$(VERSION) .

clean:
	rm -rf bin dist
