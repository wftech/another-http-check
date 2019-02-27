CWD=$(shell pwd)
APP_GID=$(shell id --group)
APP_USER=${USER}
CONTAINER_NAME=another-http-check
BIN_NAME=another-http-check

defualt: binary

test: build
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) go test

binary: build clean
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) \
		go build -ldflags '-d' -tags netgo -installsuffix netgo -o $(BIN_NAME)

runshell: build
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) sh

clean:
	rm -f $(BIN_NAME)

build:
	docker build --build-arg APP_GID=$(APP_GID) --build-arg=APP_USER=$(APP_USER) \
		-t $(CONTAINER_NAME) .
