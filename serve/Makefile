STYLES := $(wildcard assets/styles/*.scss)

all: static/styles/main.css

static/styles/main.css: $(STYLES)
	npm run sass:prod

test:
	echo "server tests not implemented"

clean:
	rm -f static/styles/*.css
