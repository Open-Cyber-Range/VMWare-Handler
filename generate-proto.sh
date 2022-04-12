protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    deployer-grpc/deployer-grpc.proto

cd deployer-grpc
go mod tidy
cd ..
