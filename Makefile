SCSS := $(wildcard server/assets/styles/*.scss)
CSS := server/static/styles/main.css

all: $(CSS)
	go build

run: $(CSS)
	go run . serve -c ./config.toml

$(CSS): $(SCSS)
	make -C server

test:
	go test ./...
	make -C server test

clean:
	rm -rf dorothy
	make -C server clean
