SHELL := bash
.ONESHELL:
.SHELLFLAGS := -eu -o pipefail -c
.DELETE_ON_ERROR:
MAKEFLAGS += --warn-undefined-variables
MAKEFLAGS += --no-builtin-rules
.POSIX:
.SUFFIXES:

UNAME := $(shell uname)
VERSION ?= $(shell git describe --tags --abbrev=0)
SHA ?= $(shell git rev-parse HEAD)
BUILD_DATE := $(shell date +%Y-%m-%d\ %H:%M:%S)


SRC_FILES= main.go
OTHER_FILES= internal/filters/filters.go \
	     internal/artifacts/artifacts.go
PKGNAME=rhc-catalog-worker
.DEFAULT_GOAL := build
ifeq ($(UNAME), Linux)
TRANSFORM:= --transform s/^\./$(PKGNAME)-$(VERSION)/ 
endif

ifeq ($(UNAME), Darwin)
TRANSFORM:= -s /^\./$(PKGNAME)-$(VERSION)/
endif

LDFLAGS :=
LDFLAGS += -X 'github.com/RedHatInsights/rhc-worker-catalog/build.Version=${VERSION}'
LDFLAGS += -X 'github.com/RedHatInsights/rhc-worker-catalog/build.Build=${BUILD_DATE}'
LDFLAGS += -X 'github.com/RedHatInsights/rhc-worker-catalog/build.Sha1=${SHA}'

BUILDFLAGS :=
ifeq ($(shell find . -name vendor), ./vendor)
BUILDFLAGS += -mod=vendor
endif

.PHONY: build
build:
	go build ${BUILDFLAGS} -ldflags="${LDFLAGS}" -o ${PKGNAME} ${SRC_FILES}

.PHONY: dist
dist:
	go mod tidy
	go mod vendor
	tar --create --gzip --exclude=.git --exclude=.vscode \
		--file /tmp/$(PKGNAME)-$(VERSION).tar.gz \
		${TRANSFORM} \
		. && mv /tmp/$(PKGNAME)-$(VERSION).tar.gz .
	rm -rf ./vendor

.PHONY: vendor
vendor:
	go mod tidy
	go mod vendor

.PHONY: test
test:
	go test -race -v . ./...

.PHONY: test_debug
test_debug:
	dlv test ${SRC_FILES} ${OTHER_FILES}

.PHONY: coverage
coverage:
	rm -rf coverage.out
	go test -coverprofile=coverage.out . ./...
	go tool cover -html=coverage.out

.PHONY: format
format:
	go fmt ${SRC_FILES}

.PHONY: run
run:
	go run ${SRC_FILES}

.PHONY: race
race:
	go run -race ${SRC_FILES}

.PHONY: debug
debug:
	dlv debug ${SRC_FILES}

.PHONY: linux
linux: 
	GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -x -o ${PKGNAME}.linux ${SRC_FILES}

.PHONY: clean
clean:
	go clean

.PHONY: lint
lint:
	golint ./...
