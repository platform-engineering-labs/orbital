export VERSION := `git describe --tags --abbrev=0 | cut -c 2-`
GITHUB := env("GITHUB_ACTIONS", "false")

default: clean build setup

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

setup:
   {{ if GITHUB != "false" { "dist/bin/ops setup" } else {""} }}

pkg: build setup
    dist/bin/ops opkg build --secure

publish: pkg
    dist/bin/ops publish --repo pel --channel stable *.opkg