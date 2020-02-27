.PHONY: build clean tool lint help

all: build

build:
	@go build -o server-sugar-app cmd/main.go

tool:
	go vet ./...; true
	gofmt -w .

lint:
	golint ./...

clean:
	rm -rf go-gin-example
	go clean -i .
