<a href="https://cr14.ee">
    <img src="assets/logos/CR14-logo.svg" alt="CR14 Logo" width="100" height="100">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://eas.ee">
    <img src="assets/logos/eas-logo.svg" alt="EAS Logo" width="100" height="100">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://taltech.ee">
    <img src="assets/logos/Taltech-logo.svg" alt="Taltech Logo" width="100" height="100">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://eeagrants.org">
    <img src="assets/logos/ng.png" alt="NG Logo" width="100" height="100">
</a>
&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;
<a href="https://ntnu.edu">
    <img src="assets/logos/NTNU-logo.svg" alt="NTNU Logo" width="100" height="100">
</a>


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

3. Executor requires a Redis configuration file containing `requirepass <redis-server-password>`, any additional configuration is up to the user. For file details see `docker-compose.yml`

### Building

Just run `make build`
