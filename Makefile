.PHONY: list

COVERAGE_PATH=./output/coverage/
COMPLEXITY_PATH=./output/complexity/
BIN_PATH=./output/bin/

all: go-nuke list go-fmt go-build go-test go-lint go-cyclo go-coverage

list:
	@echo ""
	@make -qpRr | grep -E '^[a-z].*:' | cut -d: -f1 | sort
	@echo ""

go-fmt:
	gofmt -l -w ./internal ./cmd

go-build:
	mkdir -p ${BIN_PATH}
	go build -o ${BIN_PATH} ./cmd/...

go-test:
	go test -v ./...

go-lint:
	go vet -v ./...
	golangci-lint -v run ./...

go-cyclo:
	mkdir -p ${COMPLEXITY_PATH}
	gocyclo -ignore="_test.go" . > ${COMPLEXITY_PATH}complexity.txt
	cat ${COMPLEXITY_PATH}complexity.txt

go-coverage:
	mkdir -p ${COVERAGE_PATH}
	go test ./... -coverprofile=${COVERAGE_PATH}coverage.out
	go tool cover -html ${COVERAGE_PATH}coverage.out -o ${COVERAGE_PATH}coverage.html

go-nuke:
	go clean -cache
	go clean -i ./...
	go clean
	rm rm -rf ./output || true
