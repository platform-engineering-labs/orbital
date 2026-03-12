.DEFAULT_GOAL := all

clean:
	rm -rf ./dist
	rm -rf ./tmp
	find . -name '*.opkg' -type f -delete

build:
	mkdir -p dist/
	mkdir -p tmp/
	go build -C ./cmd -o ../dist/ops

tidy:
	go mod tidy
	cd cmd; go mod tidy

all: clean build

.PHONY: clean build
