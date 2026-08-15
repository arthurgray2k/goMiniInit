.PHONY: all build test clean docker-build docker-run

BINARY_NAME=gominiinit
IMAGE_NAME=gomininit-test

all: build

build:
	go build -o $(BINARY_NAME) .

test:
	go test -v ./...

docker-build:
	podman build -t $(IMAGE_NAME) .

docker-run:
	podman run --rm -it $(IMAGE_NAME)

clean:
	rm -f $(BINARY_NAME)
