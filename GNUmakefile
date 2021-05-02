SHELL = bash

HASH_SERVER_BINARY=./hash_server

GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test

.PHONY: all
all: $(HASH_SERVER_BINARY)

$(HASH_SERVER_BINARY):
	$(GOBUILD) -o $(HASH_SERVER_BINARY) ./hash_server.go

.PHONY: clean
clean:
	rm ${HASH_SERVER_BINARY}

.PHONY: test
test:
	cd tests && chmod +x *.sh && ./multiple_connections.sh


