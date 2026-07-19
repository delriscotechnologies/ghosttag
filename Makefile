.PHONY: fmt test vet build build-linux install check clean

GO ?= go
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -buildvcs=false -trimpath -o ./bin/ghosttag ./cmd/ghosttag

build-linux:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux $(GO) build -buildvcs=false -trimpath -o ./bin/ghosttag-linux ./cmd/ghosttag

install: build
	install -d "$(DESTDIR)$(BINDIR)"
	install -m 0755 ./bin/ghosttag "$(DESTDIR)$(BINDIR)/ghosttag"

check: test vet build

clean:
	rm -rf bin
