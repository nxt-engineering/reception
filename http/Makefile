.NOTPARALLEL:
.PHONY: all clean build release get_rice

# To see output, run "make <target> VERBOSE=1"
ifndef VERBOSE
.SILENT:
endif

all build: get_rice rice-box.go
dist release: clean build

clean:
	rice clean

rice-box.go:
	rice embed-go

get_rice:
	go get github.com/GeertJohan/go.rice
	go get github.com/GeertJohan/go.rice/rice
