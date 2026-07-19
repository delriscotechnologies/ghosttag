.PHONY: bootstrap fmt test vet build check

GO := bash ./scripts/go-local.sh

bootstrap:
	bash ./scripts/bootstrap-go.sh

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

build:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux $(GO) build -buildvcs=false -trimpath -o ./bin/ghosttag ./cmd/ghosttag

check: fmt test vet build
