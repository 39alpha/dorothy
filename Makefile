SCSS := $(wildcard serve/assets/styles/*.scss)
CSS := static/styles/main.css

all: $(CSS)
	go build

run: $(CSS)
	go run . serve -c ./config.toml

static/styles/main.css: $(SCSS)
	make -C serve

test:
	go test ./...
	make -C serve test

clean:
	rm -r dorothy
	make -C serve clean
