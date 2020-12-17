SRC_FILES= main.go
OTHER_FILES= internal/filters/filters.go \
	     internal/artifacts/artifacts.go
BINARY=catalog_mqtt_client
.DEFAULT_GOAL := build

build:
	go build -o ${BINARY} ${SRC_FILES}

test:
	go test -v . ./...

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
	GOOS=linux GOARCH=arm go build -x -o catalog_worker.linux ${SRC_FILES}

clean:
	go clean

lint:
	golint ./...
