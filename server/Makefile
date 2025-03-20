STYLES := $(wildcard views/assets/styles/*.scss)

all: views/static/styles/main.css

views/static/styles/main.css: $(STYLES)
	npm run sass:prod

clean:
	rm -f views/static/styles/*.css

.PHONY: clean
