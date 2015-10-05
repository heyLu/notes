GOPATH=$(PWD)/.go

all: notes

notes: *.go
	GOPATH=$(GOPATH) go build

deps:
	GOPATH=$(GOPATH) go list -f '{{ join .Imports "\n" }}' ./... | grep -v '^_' | sort | uniq | xargs go get -v

clean:
	rm -f notes

.PHONY: all deps clean
