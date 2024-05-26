SCSS := $(wildcard server/assets/styles/*.scss)
CSS := server/static/styles/main.css

all: build

build: $(CSS)
	go build

run: $(CSS)
	go run . serve -c ./config.toml

$(CSS): $(SCSS)
	make -C server

test: test-go test-cli

test-go:
	go test ./...
	make -C server test

test-cli:
	./test/bats/bin/bats $(BATS_ARGS) test/

clean:
	rm -rf dorothy
	make -C server clean

.PHONY: clean test
