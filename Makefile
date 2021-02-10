VERSION=0.0.1
SRC_FILES= main.go
OTHER_FILES= internal/filters/filters.go \
	     internal/artifacts/artifacts.go
BINARY=rhc_catalog_worker
.DEFAULT_GOAL := build

build:
	go build -ldflags="-X 'main.Version=${VERSION}' -X main.Sha1=`git rev-parse HEAD`" -o ${BINARY} ${SRC_FILES}

test:
	go test -race -v . ./...

test_debug:
	dlv test ${SRC_FILES} ${OTHER_FILES}

coverage:
	rm -rf coverage.out
	go test -coverprofile=coverage.out . ./...
	go tool cover -html=coverage.out

format:
	go fmt ${SRC_FILES}

run:
	go run ${SRC_FILES}

race:
	go run -race ${SRC_FILES}

debug:
	dlv debug ${SRC_FILES}

linux: 
	GOOS=linux GOARCH=amd64 go build -x -o ${BINARY}.linux ${SRC_FILES}

clean:
	go clean

lint:
	golint ./...
