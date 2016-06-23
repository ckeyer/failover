PWD := $(shell pwd)

default: build image 

build:
	docker run --rm --name building -v $(PWD):/go/src/failover -e CGO_ENABLED=0 -w /go/src/failover golang:1.6 go build -o bin/failover . 
	docker run --rm --name building -v $(PWD):/go/src/failover -e CGO_ENABLED=0 -w /go/src/failover golang:1.6 go build -o bin/proxy unixproxy/main.go
	cp config.json bin/config.json

image:
	docker build -t ckeyer/failover .

clean:
	rm -rf bin

local:
	go build -o failover . 
	go build -o proxy unixproxy/main.go