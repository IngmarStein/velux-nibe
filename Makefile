GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
BINARY_NAME=velux-nibe

.phony: all test clean update

all: test $(BINARY_NAME)

$(BINARY_NAME): *.go nibe/*.go velux/*.go
	$(GOBUILD) -o $(BINARY_NAME) -v

test:
	$(GOTEST) -v ./...

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

update:
	go get -u
	go mod tidy

docker-build: $(BINARY_NAME) Dockerfile
	DOCKER_BUILDKIT=1 docker build .
