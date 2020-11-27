PHONY : build run fresh test clean pack-releases

BIN := gandalf.bin

HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git show -s --format=%ci ${HASH})
BUILD_DATE := $(shell date '+%Y-%m-%d %H:%M:%S')
VERSION := ${HASH} (${COMMIT_DATE})

STATIC := ./templates:/templates

build:
	go build -o ${BIN} -ldflags="-X 'main.buildVersion=${VERSION}' -X 'main.buildDate=${BUILD_DATE}'" ./cmd/generator/
	stuffbin -a stuff -in ${BIN} -out ${BIN} ${STATIC}

run:
	./${BIN}

fresh: clean build run

test: build
	rm -rf ./test/validations_gen.go
	./gandalf.bin -dir test -file test/validations_gen.go
	go test -v ./test

clean:
	go clean
	rm -f ${BIN}


# pack-releases runs stuffbin packing on a given list of
# binaries. This is used with goreleaser for packing
# release builds for cross-build targets.
pack-releases:
	$(foreach var,$(RELEASE_BUILDS),stuffbin -a stuff -in ${var} -out ${var} ${STATIC};)