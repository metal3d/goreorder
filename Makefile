.PHONY: dist install clean

CC=go

CUR_SHA=$(shell git log -n1 --pretty='%h')
CUR_BRANCH=$(shell git branch --show-current)
VERSION=$(shell git describe --exact-match --tags $(CUR_SHA) 2>/dev/null || echo $(CUR_BRANCH)-$(CUR_SHA))

CC_OPTS=-ldflags "-X main.version=$(VERSION)"

install:
	go install -v $(CC_OPTS) ./...

uninstall:
	go clean -r -i ./...

dist:
	mkdir -p dist
	$(MAKE) dist/goreorder-linux-amd64
	$(MAKE) dist/goreorder-darwin-amd64
	$(MAKE) dist/goreorder-windows-amd64.exe


dist/goreorder-linux-amd64:
	GOOS=linux GOARCH=amd64 go build $(CC_OPTS) -o $< $(CC)

dist/goreorder-darwin-amd64:
	GOOS=darwin GOARCH=amd64 go build $(CC_OPTS) -o $< $(CC)

dist/goreorder-windows-amd64.exe:
	GOOS=windows GOARCH=amd64 go build $(CC_OPTS) -o $@ $(CC)

dist/goreorder-freebsd-amd64:
	GOOS=freebsd GOARCH=amd64 go build $(CC_OPTS) -o $< $(CC)


clean:
	rm -rf dist

