VERSION := `git describe --tags --abbrev=0`
BUILD_DATE := `date +%Y-%m-%d\ %H:%M:%S`
SHA := `git rev-parse HEAD`

SRC_FILES= main.go
OTHER_FILES= internal/filters/filters.go \
	     internal/artifacts/artifacts.go
BINARY=rhc_catalog_worker
.DEFAULT_GOAL := build

LDFLAGS :=
LDFLAGS += -X 'github.com/RedHatInsights/rhc-worker-catalog/build.Version=${VERSION}'
LDFLAGS += -X 'github.com/RedHatInsights/rhc-worker-catalog/build.Build=${BUILD_DATE}'
LDFLAGS += -X 'github.com/RedHatInsights/rhc-worker-catalog/build.Sha1=${SHA}'
build:
	go build -ldflags="${LDFLAGS}" -o ${BINARY} ${SRC_FILES}

vendor:
	go mod tidy
	go mod vendor
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
	GOOS=linux GOARCH=amd64 go build -ldflags="${LDFLAGS}" -x -o ${BINARY}.linux ${SRC_FILES}

clean:
	go clean

lint:
	golint ./...
