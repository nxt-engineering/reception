COMMIT=`git show -s --format=%h`
DATE=`date -u +%FT%T%z`
TAG=`git describe --abbrev=0 --tags 2>/dev/null || echo "0.0.0"`
BRANCH=git rev-parse --abbrev-ref HEAD
LDFLAGS=-ldflags "-w -s -X main.Branch=`${BRANCH}` -X main.Tag=${TAG} -X main.Commit=${COMMIT} -X main.BuildDate=${DATE}"

all: reception
release: clean release-tag all
	echo ""
	./reception -v

reception:
	$(MAKE) -C http
	go build ${LDFLAGS} reception.go

clean:
	$(MAKE) -C http clean
	rm -f reception

run:
	go run reception.go

ifdef VERSION
release-tag:
	# only tag on master branch!
	$(BRANCH) | grep -e "^master$$" > /dev/null \
		|| ( echo "Releases should only be made from master branch."; false )

	git diff-index --quiet HEAD -- \
		|| ( echo "There are uncommitted changes. Releases show only be made on clean HEADs."; false )

	git tag ${VERSION}
	echo "Don't forget to push the new tag!"
else
release-tag:
	echo "You have not defined a VERSION. Run \"make release VERSION=1.2.3\" to set a version."; false
endif

.PHONY: clean run release release-tag

# To see output, run "make <target> VERBOSE=1"
ifndef VERBOSE
.SILENT:
endif
