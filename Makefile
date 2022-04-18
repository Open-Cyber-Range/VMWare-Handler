compile-protobuf:
	protoc --go_out=grpc --go-grpc_out=grpc \
	--go_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/common \
	--go_opt=Msrc/node.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/node \
	--go-grpc_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/common \
	--go-grpc_opt=Msrc/node.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/node  \
	--go_opt=module=github.com/open-cyber-range/vmware-node-deployer/grpc \
	--go-grpc_opt=module=github.com/open-cyber-range/vmware-node-deployer/grpc \
	--proto_path=grpc/proto src/node.proto src/common.proto

build: compile-protobuf
	go build -o ./bin/wmvare-node-deployer ./deployer

test: build
	go test -v ./deployer

run: build
	./bin/wmvare-node-deployer ./config.yml

test-and-build: test build
