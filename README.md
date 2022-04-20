## VMWare Node Deployer

This is a microservice for deploying nodes defined in the SDL to the VMWare environment.

## Development

### Requirements

1. Protobuf-compiler

`sudo apt install protobuf-compiler`

2. Go protobuf and gRPC plugins

```
go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.28
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.2
```

### Building

Just run `make build`
