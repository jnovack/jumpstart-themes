LDFLAGS = -X main.version=$(VERSION) \
          -X main.revision=$(REVISION) \
          -X main.buildRFC3339=$(BUILD_DATE)

.PHONY: build clean test vet

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(APPLICATION) ./cmd/$(APPLICATION)

test:
	go test ./...

vet:
	go vet ./...

clean:
	rm -f bin/$(APPLICATION)
	go clean ./...
