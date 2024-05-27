SOURCES=$(wildcard *.adoc)

all: index.html

index.html: ../README.adoc $(SOURCES)
	asciidoctor -b html5 index.adoc

clean:
	rm -rf *.html

.PHONY: clean
