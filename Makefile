.PHONY: all
all: fmt lint vet coverage

.PHONY: build
build:
	go build -o akamai cli/main.go

.PHONY: test
test:
	go test -count=1 ./...

.PHONY: coverage-ui
coverage-ui:
	go test -covermode=count -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

.PHONY: coverage
coverage:
	go test -coverprofile coverage.out ./...
	go tool cover -func coverage.out | grep total

.PHONY: lint
lint:
	golint -set_exit_status ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: release
release:
	./build.sh


.PHONY: pack
pack:
	tar -zcvf cli.tar.gz .