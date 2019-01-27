.PHONY: build lint vet compile

default: build

build: vet lint compile

compile:
	go build -p 16 .

lint:
	golint .

vet:
	go vet ./...

linux:
	@$(MAKE) GOOS=linux GOARCH=amd64
