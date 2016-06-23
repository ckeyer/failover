PWD := $(shell pwd)

binary:
	docker run --rm --name building -v $(PWD):/go/src/failover -w /go/src/failover golang:1.6 go build .