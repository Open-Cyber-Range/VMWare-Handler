.PHONY: all clean install compile-protobuf build build-machiner build-switcher test test-machiner test-switcher run-machiner run-switcher test-and-build build-deb generate-nsx-t-openapi

all: build

clean:
	rm -f bin/ranger-vmware-*
	rm -f ../*.deb

clean-test-cache:
	go clean -testcache

install:
	mkdir -p ${DESTDIR}/var/opt/ranger/bin
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-machiner
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-switcher
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-templater
	mkdir -p ${DESTDIR}/lib/systemd/system/
	mkdir -p ${DESTDIR}/etc/systemd/system/
	cp bin/ranger-vmware-machiner $(DESTDIR)/var/opt/ranger/bin/
	cp bin/ranger-vmware-switcher $(DESTDIR)/var/opt/ranger/bin/
	cp bin/ranger-vmware-templater $(DESTDIR)/var/opt/ranger/bin/
	cp extra/machiner-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-machiner/config.yml
	cp extra/switcher-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-switcher/config.yml
	cp extra/templater-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-switcher/config.yml
	cp extra/ranger-vmware-machiner.service $(DESTDIR)/lib/systemd/system/
	cp extra/ranger-vmware-switcher.service $(DESTDIR)/lib/systemd/system/
	cp extra/ranger-vmware-templater.service $(DESTDIR)/lib/systemd/system/
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-machiner.service $(DESTDIR)/etc/systemd/system/ranger-vmware-machiner.service
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-switcher.service $(DESTDIR)/etc/systemd/system/ranger-vmware-switcher.service
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-templater.service $(DESTDIR)/etc/systemd/system/ranger-vmware-templater.service

compile-protobuf:
	protoc --go_out=grpc --go-grpc_out=grpc \
	--go_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-handler/grpc/common \
	--go_opt=Msrc/template.proto=github.com/open-cyber-range/vmware-handler/grpc/template \
	--go_opt=Msrc/capability.proto=github.com/open-cyber-range/vmware-handler/grpc/capability \
	--go_opt=Msrc/virtual-machine.proto=github.com/open-cyber-range/vmware-handler/grpc/virtual-machine \
	--go_opt=Msrc/switch.proto=github.com/open-cyber-range/vmware-handler/grpc/switch \
	--go_opt=Msrc/feature.proto=github.com/open-cyber-range/vmware-handler/grpc/feature \
	--go-grpc_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-handler/grpc/common \
	--go-grpc_opt=Msrc/template.proto=github.com/open-cyber-range/vmware-handler/grpc/template \
	--go-grpc_opt=Msrc/capability.proto=github.com/open-cyber-range/vmware-handler/grpc/capability  \
	--go-grpc_opt=Msrc/virtual-machine.proto=github.com/open-cyber-range/vmware-handler/grpc/virtual-machine  \
	--go-grpc_opt=Msrc/switch.proto=github.com/open-cyber-range/vmware-handler/grpc/switch  \
	--go-grpc_opt=Msrc/feature.proto=github.com/open-cyber-range/vmware-handler/grpc/feature  \
	--go_opt=module=github.com/open-cyber-range/vmware-handler/grpc \
	--go-grpc_opt=module=github.com/open-cyber-range/vmware-handler/grpc \
	--proto_path=grpc/proto src/virtual-machine.proto src/switch.proto src/common.proto src/capability.proto src/template.proto src/feature.proto

generate-nsx-t-openapi:
	java -Dapis=Segments,Connectivity -Dmodels -DsupportingFiles -jar /var/opt/swagger/swagger-codegen-cli.jar generate -DpackageName=nsx_t_openapi -DmodelTests=false -DapiTests=false -DapiDocs=false -DmodelDocs=false -D io.swagger.parser.util.RemoteUrl.trustAll=true -i extra/nsx_policy_api.yaml -l go -o nsx_t_openapi &&\
	find ./nsx_t_openapi -type f ! \( -name '*.go' \) -exec rm -f {} +

build: compile-protobuf build-machiner build-switcher build-templater

build-machiner: compile-protobuf
	go build -o bin/ranger-vmware-machiner ./machiner

build-switcher: compile-protobuf
	go build -o bin/ranger-vmware-switcher ./switcher

build-templater: compile-protobuf
	go build -o bin/ranger-vmware-templater ./templater


test: build test-machiner test-switcher test-templater

test-machiner: build-machiner
	go test -v ./machiner

test-switcher: build-switcher
	go test -v ./switcher

test-templater: build-templater
	go test -v ./templater

test-library:
	go test -v ./library

run-machiner: build-machiner
	bin/ranger-vmware-machiner machiner-config.yml

run-switcher: build-switcher
	bin/ranger-vmware-switcher switcher-config.yml

run-templater: build-templater
	bin/ranger-vmware-templater

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

build-deb: clean build
	env DH_VERBOSE=1 dpkg-buildpackage -b --no-sign
