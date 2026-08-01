.PHONY: run build tidy test vet fmt

run:
	go run .

build:
	go build -o fsldk-api.exe .

tidy:
	go mod tidy

vet:
	go vet ./...

fmt:
	go fmt ./...

test:
	go test ./...
