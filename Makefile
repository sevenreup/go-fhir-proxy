build:
	@go build -C cmd/proxy -o ../../bin/proxy

run: build
	@./bin/proxy

test:
	@go test -v ./...