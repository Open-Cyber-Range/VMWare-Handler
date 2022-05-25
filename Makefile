.PHONY: all clean install compile-protobuf build test test-and-build build-deb

all: build

clean:
	rm -f bin/ranger-vmware-node-deployer

install:
	mkdir -p ${DESTDIR}/var/opt/ranger/bin
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-node-deployer
	mkdir -p ${DESTDIR}/lib/systemd/system/
	mkdir -p ${DESTDIR}/etc/systemd/system/
	cp bin/ranger-vmware-node-deployer $(DESTDIR)/var/opt/ranger/bin/
	cp extra/example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-node-deployer/config.yml
	cp extra/ranger-vmware-node-deployer.service $(DESTDIR)/lib/systemd/system/
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-node-deployer.service $(DESTDIR)/etc/systemd/system/ranger-vmware-node-deployer.service

compile-protobuf:
	protoc --go_out=grpc --go-grpc_out=grpc \
	--go_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/common \
	--go_opt=Msrc/node.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/node \
	--go-grpc_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/common \
	--go-grpc_opt=Msrc/node.proto=github.com/open-cyber-range/vmware-node-deployer/grpc/node  \
	--go_opt=module=github.com/open-cyber-range/vmware-node-deployer/grpc \
	--go-grpc_opt=module=github.com/open-cyber-range/vmware-node-deployer/grpc \
	--proto_path=grpc/proto src/node.proto src/common.proto

build: compile-protobuf build-machiner build-switcher

build-machiner: compile-protobuf
	go build -o bin/ranger-vmware-node-deployer ./machiner

build-switcher: compile-protobuf
	go build -o bin/ranger-vmware-node-deployer ./switcher

test: build test-deployer test-switcher

test-deployer: build-machiner
	go test -v ./machiner

test-switcher: build-switcher
	go test -v ./switcher

run: build
	bin/ranger-vmware-node-deployer config.yml

test-and-build: test build

uninstall:
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-node-deployer
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-node-deployer/config.yml
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-node-deployer.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-node-deployer.service

build-deb:
	env DH_VERBOSE=1 dpkg-buildpackage -b --no-sign
