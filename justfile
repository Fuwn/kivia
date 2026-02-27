set shell := ["bash", "-eu", "-o", "pipefail", "-c"]

build:
	mkdir -p bin
	go build -o ./bin/kivia .

format:
	iku -w . || go fmt ./...

test:
	go test ./...

run *args:
	go run . {{args}}

self:
	go run . --path ./... --ignore file=testdata

install:
	go install .
