export VERSION := `git describe --tags --abbrev=0 | cut -c 2-`

default: clean build

clean:
	rm -rf ./dist
	rm -rf ./tmp
	find . -name '*.opkg' -type f -delete

build:
	mkdir -p dist/bin
	mkdir -p tmp/
	go build -C ./cmd -o ../dist/bin/ops

tidy:
	go mod tidy
	cd cmd; go mod tidy

pkg: build
    dist/bin/ops opkg build --secure