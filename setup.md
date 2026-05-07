# Commands to run

```
# Check where go is installed
$ go env GOPATH

# Install proto related binaries
$ go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
$ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Check installed binaries in bin
$ ls "$(go env GOPATH)/bin" | grep protoc-gen

# Below command adds the bin directory to your PATH variable, so you can run the installed binaries from anywhere in the terminal
$ echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
$ source ~/.zshrc

$ which protoc-gen-go
--> ~/go/bin/protoc-gen-go

$ which protoc-gen-go-grpc
--> ~/go/bin/protoc-gen-go-grpc

$ protoc-gen-go --version
---> protoc-gen-go v1.36.11

$ protoc-gen-go-grpc --version
---> protoc-gen-go-grpc 1.6.1

# After creating proto files run the below command to generate go files + grpc server files
$ make proto
--> Generates *.pb.go files under proto/search

# update go.mod file for imports
$ go mod tidy

# ./... means to build all the packages in the current directory and its subdirectories
$ go build ./...

```


BadgerDB integration

```
$ go get github.com/dgraph-io/badger/v4
```