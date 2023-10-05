BINARY_NAME=for-each
VERSION=$(shell git describe --tags --always --dirty)
PLATFORMS := windows linux darwin

os = $(word 1, $@)

build: deps
	go build

deps:
	go mod download

all: deps windows linux darwin

$(PLATFORMS):
	GOOS=$(os) GOARCH=amd64 go build
	mkdir -p build/$(VERSION)/$(os)/
	mv $(BINARY_NAME)$(if $(filter $(os),windows),.exe,) build/$(VERSION)/$(os)/
	if [ $(os) = "windows" ]; then \
		cd build/$(VERSION)/$(os) && zip ../../$(BINARY_NAME)-$(VERSION)-$(os)-amd64.zip $(BINARY_NAME).exe; \
	else \
		tar czvf build/$(BINARY_NAME)-$(VERSION)-$(os)-amd64.tar.gz -C build/$(VERSION)/$(os)/ $(BINARY_NAME); \
	fi

clean:
	rm -rf build/
	rm -f *.tar.gz
	rm -f *.zip

.PHONY: clean build deps all $(PLATFORMS)
