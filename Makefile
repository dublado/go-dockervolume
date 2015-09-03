.PHONY: \
	all \
	deps \
	updatedeps \
	testdeps \
	updatetestdeps \
	build \
	lint \
	vet \
	errcheck \
	pretest \
	test \
	clean \
	proto

all: test

deps:
	go get -d -v ./...

updatedeps:
	go get -d -v -u -f ./...

testdeps:
	go get -d -v -t ./...

updatetestdeps:
	go get -d -v -t -u -f ./...

build: deps
	GOOS=linux go build ./...

lint: testdeps
	go get -v github.com/golang/lint/golint
	for file in $$(find . -name '*.go'); do \
		golint $$file; \
		if [ -n "$$(golint $$file)" ]; then \
			exit 1; \
		fi; \
	done

vet: testdeps
	-go get -v golang.org/x/tools/cmd/vet
	go vet ./...

errcheck: testdeps
	go get -v github.com/kisielk/errcheck
	errcheck ./...

pretest: lint vet errcheck

test: testdeps pretest
	GOOS=linux go test -test.v ./...

clean:
	go clean ./...
	GOOS=linux go clean ./...

proto:
	go get -v github.com/peter-edge/go-tools/docker-protoc-all
	docker pull pedge/protolog
	docker-protoc-all go.pedge.io/dockervolume
	rm /tmp/protolog.pb.go
	tail -n +$$(grep -n 'package dockervolume' protolog.pb.go | cut -f 1 -d :) protolog.pb.go > /tmp/protolog.pb.go
	/tmp/protolog.pb.go protolog.pb.go
