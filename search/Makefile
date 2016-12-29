# Borrowed from: 
# https://github.com/silven/go-example/blob/master/Makefile
# https://vic.demuzere.be/articles/golang-makefile-crosscompile/

BINARY = biblescholar
VET_REPORT = vet.report
TEST_REPORT = tests.xml
GOARCH = amd64

COMMIT=$(shell git rev-parse HEAD)
BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
GITHUB_USERNAME=turtlemonvh

# For symlinks
PROJECT_BASE_DIR=$(shell dirname $$(pwd))
GOPATH_BASE_DIR=${GOPATH}/src/github.com/${GITHUB_USERNAME}/biblescholar
GOPATH_BASE_DIR_LINK=$(shell readlink ${GOPATH_BASE_DIR})

# Core path variables
BUILD_DIR=${GOPATH}/src/github.com/${GITHUB_USERNAME}/biblescholar/search
BINARY_BUILD_DIR=${BUILD_DIR}/command
ARTIFACT_DIR=${BUILD_DIR}/artifacts

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X main.buildCommit=${COMMIT} -X main.buildBranch=${BRANCH}"

# Build the project
all: link clean test vet linux darwin windows

link:
	PROJECT_BASE_DIR=${PROJECT_BASE_DIR} ; \
	GOPATH_BASE_DIR=${GOPATH_BASE_DIR} ; \
	GOPATH_BASE_DIR_LINK=${GOPATH_BASE_DIR_LINK} ; \
	if [ "$${GOPATH_BASE_DIR_LINK}" != "$${PROJECT_BASE_DIR}" ] ; then \
	    echo "Fixing symlinks for build" ; \
	    rm -f $${GOPATH_BASE_DIR} ; \
	    ln -s $${PROJECT_BASE_DIR} $${GOPATH_BASE_DIR} ; \
	fi

artifactdir:
	mkdir -p ${ARTIFACT_DIR}

linux: link artifactdir
	cd ${BINARY_BUILD_DIR}; \
	GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} . ; \
	mv ${BINARY}-linux-${GOARCH} ${ARTIFACT_DIR}/ ; \
	cd - >/dev/null

darwin: link artifactdir
	cd ${BINARY_BUILD_DIR}; \
	GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-darwin-${GOARCH} . ; \
	mv ${BINARY}-darwin-${GOARCH} ${ARTIFACT_DIR}/ ; \
	cd - >/dev/null

windows: link artifactdir
	cd ${BINARY_BUILD_DIR}; \
	GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-windows-${GOARCH}.exe . ; \
	mv ${BINARY}-windows-${GOARCH}.exe ${ARTIFACT_DIR}/ ; \
	cd - >/dev/null

test: artifactdir
	if ! hash go2xunit 2>/dev/null; then go install github.com/tebeka/go2xunit; fi
	cd ${BUILD_DIR}; \
	go test -v ./... 2>&1 | go2xunit -output ${TEST_REPORT} ; \
	mv ${TEST_REPORT} ${ARTIFACT_DIR}/ ; \
	cd - >/dev/null

vet: artifactdir
	-cd ${BUILD_DIR}; \
	go vet ./... > ${VET_REPORT} 2>&1 ; \
	mv ${VET_REPORT} ${ARTIFACT_DIR}/ ; \
	cd - >/dev/null

fmt:
	cd ${BUILD_DIR}; \
	go fmt $$(go list ./... | grep -v /vendor/) ; \
	cd - >/dev/null

clean:
	-rm -rf ${ARTIFACT_DIR}

.PHONY: link linux darwin windows test vet fmt clean