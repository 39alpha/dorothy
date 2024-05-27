SCSS := $(wildcard server/assets/styles/*.scss)
CSS := server/static/styles/main.css

all: build

build: $(CSS)
	go build

run: $(CSS)
	go run . serve -c ./config.toml

docs:
	make -C docs

$(CSS): $(SCSS)
	make -C server

test: test-go test-cli

test-go:
	@echo Running Go Tests
	@mkdir -p coverage
	@go test -count=1 -cover -coverprofile=coverage/c.out -covermode=atomic ./...
	@go tool cover -html=coverage/c.out -o=coverage/index.html

test-cli:
	@echo Running CLI Tests
	@./test/bats/bin/bats $(BATS_ARGS) test/

clean:
	rm -rf dorothy coverage
	make -C server clean

.PHONY: clean test docs
