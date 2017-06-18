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

# Core path variables
BUILD_DIR=${GOPATH}/src/github.com/${GITHUB_USERNAME}/biblescholar/search
BINARY_BUILD_DIR=${BUILD_DIR}/command
ARTIFACT_DIR=${BUILD_DIR}/artifacts

# Setup the -ldflags option for go build here, interpolate the variable values
LDFLAGS = -ldflags "-X main.buildCommit=${COMMIT} -X main.buildBranch=${BRANCH}"

# Build the project
all: clean test vet linux darwin windows elbzip

artifactdir:
	mkdir -p ${ARTIFACT_DIR}

linux: artifactdir
	GOOS=linux GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-linux-${GOARCH} .
	mv ${BINARY}-linux-${GOARCH} ${ARTIFACT_DIR}/ 

darwin: artifactdir
	cd command ; \
	GOOS=darwin GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-darwin-${GOARCH} .  ; \
	mv ${BINARY}-darwin-${GOARCH} ${ARTIFACT_DIR}/  ; \
	cd --

windows: artifactdir
	GOOS=windows GOARCH=${GOARCH} go build ${LDFLAGS} -o ${BINARY}-windows-${GOARCH}.exe . 
	mv ${BINARY}-windows-${GOARCH}.exe ${ARTIFACT_DIR}/

elbzip: linux
	# Make elasticbeanstalk artifact
	# https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/applications-sourcebundle.html
	rm -f ${ARTIFACT_DIR}/elb.zip
	cp ${ARTIFACT_DIR}/${BINARY}-linux-${GOARCH} .
	zip -r ${ARTIFACT_DIR}/elb.zip ${BINARY}-linux-${GOARCH} verses.bleve Dockerfile
	# Check contents
	unzip -l ${ARTIFACT_DIR}/elb.zip

test: artifactdir
	if ! hash go2xunit 2>/dev/null; then go install github.com/tebeka/go2xunit; fi
	go test -v ./... 2>&1 | go2xunit -output ${TEST_REPORT} 
	mv ${TEST_REPORT} ${ARTIFACT_DIR}/ 

vet: artifactdir
	go vet ./... > ${VET_REPORT} 2>&1 
	mv ${VET_REPORT} ${ARTIFACT_DIR}/ 

fmt:
	go fmt $$(go list ./... | grep -v /vendor/) 

clean:
	-rm -rf ${ARTIFACT_DIR}

.PHONY: linux darwin windows test vet fmt clean elbzip
