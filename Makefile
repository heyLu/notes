GOPATH=$(PWD)/.go

all: notes

notes: *.go
	GOPATH=$(GOPATH) go build

deps:
	GOPATH=$(GOPATH) go get -v ./...

clean:
	rm -f notes

.PHONY: all deps clean
