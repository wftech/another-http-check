CWD=$(shell pwd)
APP_GID=$(shell id --group)
APP_USER=${USER}
CONTAINER_NAME=another-http-check
BIN_NAME=another-http-check
RPM_SPEC_NAME=another-http-check.spec
GO_VERSION=$(shell grep FROM Dockerfile | awk '{ print $2 }' | sed 's/[a-z:-]//g' | xargs echo -n)
APP_VERSION=$(shell date +"%Y%m%d")

default: binary

test: build
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) \
		go test -v -ldflags "-X main.goVersion=$(GO_VERSION) -X main.appVersion=$(APP_VERSION)"

binary: build clean
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) \
		go build -ldflags "-d -s -w -X main.goVersion=$(GO_VERSION) -X main.appVersion=$(APP_VERSION)" \
		-tags netgo -installsuffix netgo -o $(BIN_NAME)

rpm: binary
	rm -rf rpmbuild
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) \
		rpmbuild -ba $(RPM_SPEC_NAME)
	cp rpmbuild/RPMS/x86_64/*.rpm .

runshell: build
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) sh

clean:
	rm -f $(BIN_NAME)

upgrade-dependencies: build
	docker run -v $(CWD):/app -it --rm $(CONTAINER_NAME) go get -u

build:
	docker build --build-arg APP_GID=$(APP_GID) --build-arg=APP_USER=$(APP_USER) \
		-t $(CONTAINER_NAME) .
