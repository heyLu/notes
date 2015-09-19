GOPATH=$(PWD)/.go

all: notes

notes: notes.go
	GOPATH=$(GOPATH) go build notes.go

deps:
	GOPATH=$(GOPATH) go get -v ./...

clean:
	rm -f notes

.PHONY: all deps clean
