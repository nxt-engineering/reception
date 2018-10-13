VERSION := `cat VERSION | tr -d '\n'`
COMMIT  := `git show -s --format=%h`
DATE    := `date -u +%FT%T%z`
TAG     := `git describe --abbrev=0 --tags 2>/dev/null || echo "0.0.0"`
BRANCH  := git rev-parse --abbrev-ref HEAD
LDFLAGS  = -ldflags "-w -s -X main.Branch=`${BRANCH}` -X main.Tag=${TAG} -X main.Commit=${COMMIT} -X main.BuildDate=${DATE}"

GOCMD     := go
GOBUILD   := $(GOCMD) build
GOCLEAN   := $(GOCMD) clean
GOGET     := $(GOCMD) get
GORUN     := $(GOCMD) run
GOTEST    := $(GOCMD) test

BINARY_NAME := reception

# run all these commands on the Makefiles in the SUBDIRS as well
SUBDIRTARGETS := all dist build clean
SUBDIRS       := http

DIST_DIR := dist/
DISTS    := $(DIST_DIR)${BINARY_NAME}_windows_x86.exe $(DIST_DIR)${BINARY_NAME}_windows_x64.exe $(DIST_DIR)${BINARY_NAME}_linux_x86 $(DIST_DIR)${BINARY_NAME}_linux_x64 $(DIST_DIR)${BINARY_NAME}_linux_arm64 $(DIST_DIR)${BINARY_NAME}_mac_x86 $(DIST_DIR)${BINARY_NAME}_mac_x64

# To see output, run "make <target> VERBOSE=1"
ifndef VERBOSE
.SILENT:
endif

.PHONY: clean run release tag build dist $(SUBDIRS) get-windows-dependencies install uninstall as_root

all: build

as_root:
	test "$$(id -u)" -eq "0" || ( echo "Please run 'make $(MAKECMDGOALS)' as root."; return 1 )

install: as_root
	systemctl stop reception || true
	cp reception /usr/bin/
	cp contrib/reception.service /etc/systemd/system/
	systemctl daemon-reload
	systemctl enable reception
	systemctl restart reception

uninstall: as_root
	systemctl stop reception || true
	systemctl disable reception || true
	rm /usr/bin/reception || true
	rm /etc/systemd/system/reception.service || true
	systemctl daemon-reload

# remove the mess created by make
clean:
	$(GOCLEAN)
	rm -rf $(DIST_DIR)

run: ; $(GORUN) reception.go

build:
	$(GOGET)
	$(GOBUILD) ${LDFLAGS} -o ${BINARY_NAME} reception.go

# cut a release
release: tag clean dist
	echo ""
	echo "Don't forget to create a new release on Github!"
	echo ""

# build all variants
dist: $(DISTS)
	cd $(DIST_DIR); \
	gzip *; \
	sha256sum * > sha256sum.txt; \
	cat sha256sum.txt

# ensure, that the dist dir exists
dist-dir: ; mkdir -p $(DIST_DIR)
$(DISTS): dist-dir

# windows builds
get-windows-dependencies:
	go get github.com/Microsoft/go-winio
	go get golang.org/x/sys/windows

$(DIST_DIR)${BINARY_NAME}_windows_x86.exe: get-windows-dependencies
	GOOS=windows GOARCH=386 GO386=sse2 $(GOBUILD) ${LDFLAGS} -o $@ reception.go
$(DIST_DIR)${BINARY_NAME}_windows_x64.exe: get-windows-dependencies
	GOOS=windows GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $@ reception.go

# linux builds
$(DIST_DIR)${BINARY_NAME}_linux_x86:
	GOOS=linux GOARCH=386 GO386=sse2 $(GOBUILD) ${LDFLAGS} -o $@ reception.go
$(DIST_DIR)${BINARY_NAME}_linux_x64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $@ reception.go
$(DIST_DIR)${BINARY_NAME}_linux_arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) ${LDFLAGS} -o $@ reception.go

# mac builds
$(DIST_DIR)${BINARY_NAME}_mac_x86:
	GOOS=darwin GOARCH=386 GO386=sse2 $(GOBUILD) ${LDFLAGS} -o $@ reception.go
$(DIST_DIR)${BINARY_NAME}_mac_x64:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) ${LDFLAGS} -o $@ reception.go

# run any command defined in SUBDIRTARGETS also for any Makefile in the defined SUBDIRS
$(SUBDIRTARGETS): $(SUBDIRS)
$(SUBDIRS): ; $(MAKE) -C $@ $(MAKECMDGOALS)

tag:
	# only tag on master branch!
	$(BRANCH) | grep -e "^master$$" > /dev/null \
		|| ( echo "Releases should only be made from master branch."; false )

	git diff-index --quiet HEAD -- \
		|| ( echo "There are uncommitted changes. Releases show only be made on clean HEADs."; false )

	git tag ${VERSION}

	echo ""
	echo "Don't forget to push the new tag!"
	echo ""
