# builds the Cli and Svc1 & Svc2 projects

all: clean
	@GO111MODULE=off go get -v ./...
	@go build -v -o build/svc1 ./svc1/cmd/main.go
	@go build -v -o build/svc2 ./svc2/cmd/main.go
	@go build -v -o build/cli  ./cli/main.go

clean:
	@rm -rf build
