.PHONY: all clean install compile-protobuf build build-machiner build-switcher test test-machiner test-switcher run-machiner run-switcher test-and-build build-deb

all: build

clean:
	rm -f bin/ranger-vmware-handler

install:
	mkdir -p ${DESTDIR}/var/opt/ranger/bin
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-machiner
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-switcher
	mkdir -p ${DESTDIR}/lib/systemd/system/
	mkdir -p ${DESTDIR}/etc/systemd/system/
	cp bin/ranger-vmware-machiner $(DESTDIR)/var/opt/ranger/bin/
	cp bin/ranger-vmware-switcher $(DESTDIR)/var/opt/ranger/bin/
	cp extra/machiner-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-machiner/config.yml
	cp extra/switcher-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-switcher/config.yml
	cp extra/ranger-vmware-machiner.service $(DESTDIR)/lib/systemd/system/
	cp extra/ranger-vmware-switcher.service $(DESTDIR)/lib/systemd/system/
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-machiner.service $(DESTDIR)/etc/systemd/system/ranger-vmware-machiner.service
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-switcher.service $(DESTDIR)/etc/systemd/system/ranger-vmware-switcher.service

compile-protobuf:
	protoc --go_out=grpc --go-grpc_out=grpc \
	--go_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-handler/grpc/common \
	--go_opt=Msrc/node.proto=github.com/open-cyber-range/vmware-handler/grpc/node \
	--go-grpc_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-handler/grpc/common \
	--go-grpc_opt=Msrc/node.proto=github.com/open-cyber-range/vmware-handler/grpc/node  \
	--go_opt=module=github.com/open-cyber-range/vmware-handler/grpc \
	--go-grpc_opt=module=github.com/open-cyber-range/vmware-handler/grpc \
	--proto_path=grpc/proto src/node.proto src/common.proto

build: compile-protobuf build-machiner build-switcher

build-machiner: compile-protobuf
	go build -o bin/ranger-vmware-machiner ./machiner

build-switcher: compile-protobuf
	go build -o bin/ranger-vmware-switcher ./switcher

test: build test-machiner test-switcher

test-machiner: build-machiner
	go test -v ./machiner

test-switcher: build-switcher
	go test -v ./switcher

run-machiner: build
	bin/ranger-vmware-machiner machiner-config.yml

run-switcher: build
	bin/ranger-vmware-switcher switcher-config.yml

test-and-build: test build

uninstall:
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-machiner
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-switcher
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-machiner/config.yml
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-switcher/config.yml
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-machiner.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-machiner.service
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-switcher.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-switcher.service

build-deb:
	env DH_VERBOSE=1 dpkg-buildpackage -b --no-sign
