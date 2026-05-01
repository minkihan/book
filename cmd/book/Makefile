.PHONY: build test lint tidy clean install

build:
	go build -o book .

test:
	go test -v ./...

lint:
	go fmt ./...
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -f book

install: build
	cp book ~/go/bin/book
