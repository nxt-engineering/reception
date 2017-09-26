.PHONY: all clean build release

all build: rice-box.go
dist release: clean build

clean:
	rice clean

rice-box.go:
	rice embed-go
