.PHONY: all clean install compile-protobuf build build-machiner build-switcher build-templater build-executor build-general test test-machiner test-switcher test-templater test-executor test-general test-permissions test-environment test-windows run-machiner run-switcher run-executor run-general test-injects test-and-build build-deb generate-nsx-t-openapi

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
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-executor
	mkdir -p ${DESTDIR}/etc/opt/ranger/ranger-vmware-general
	mkdir -p ${DESTDIR}/lib/systemd/system/
	mkdir -p ${DESTDIR}/etc/systemd/system/
	
	cp bin/ranger-vmware-machiner $(DESTDIR)/var/opt/ranger/bin/
	cp bin/ranger-vmware-switcher $(DESTDIR)/var/opt/ranger/bin/
	cp bin/ranger-vmware-templater $(DESTDIR)/var/opt/ranger/bin/
	cp bin/ranger-vmware-executor $(DESTDIR)/var/opt/ranger/bin/
	cp bin/ranger-vmware-general $(DESTDIR)/var/opt/ranger/bin/

	cp extra/machiner-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-machiner/config.yml
	cp extra/switcher-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-switcher/config.yml
	cp extra/templater-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-switcher/config.yml
	cp extra/executor-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-executor/config.yml
	cp extra/general-example-config.yml $(DESTDIR)/etc/opt/ranger/ranger-vmware-executor/config.yml
	
	cp extra/ranger-vmware-machiner.service $(DESTDIR)/lib/systemd/system/
	cp extra/ranger-vmware-switcher.service $(DESTDIR)/lib/systemd/system/
	cp extra/ranger-vmware-templater.service $(DESTDIR)/lib/systemd/system/
	cp extra/ranger-vmware-executor.service $(DESTDIR)/lib/systemd/system/
	cp extra/ranger-vmware-general.service $(DESTDIR)/lib/systemd/system/

	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-machiner.service $(DESTDIR)/etc/systemd/system/ranger-vmware-machiner.service
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-switcher.service $(DESTDIR)/etc/systemd/system/ranger-vmware-switcher.service
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-templater.service $(DESTDIR)/etc/systemd/system/ranger-vmware-templater.service
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-executor.service $(DESTDIR)/etc/systemd/system/ranger-vmware-executor.service
	ln -sf $(DESTDIR)/lib/systemd/system/ranger-vmware-general.service $(DESTDIR)/etc/systemd/system/ranger-vmware-general.service

compile-protobuf:
	protoc --go_out=grpc --go-grpc_out=grpc \
	--go_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-handler/grpc/common \
	--go_opt=Msrc/template.proto=github.com/open-cyber-range/vmware-handler/grpc/template \
	--go_opt=Msrc/capability.proto=github.com/open-cyber-range/vmware-handler/grpc/capability \
	--go_opt=Msrc/virtual-machine.proto=github.com/open-cyber-range/vmware-handler/grpc/virtual-machine \
	--go_opt=Msrc/switch.proto=github.com/open-cyber-range/vmware-handler/grpc/switch \
	--go_opt=Msrc/feature.proto=github.com/open-cyber-range/vmware-handler/grpc/feature \
	--go_opt=Msrc/condition.proto=github.com/open-cyber-range/vmware-handler/grpc/condition \
	--go_opt=Msrc/inject.proto=github.com/open-cyber-range/vmware-handler/grpc/inject \
	--go_opt=Msrc/event.proto=github.com/open-cyber-range/vmware-handler/grpc/event \
	--go_opt=Msrc/deputy.proto=github.com/open-cyber-range/vmware-handler/grpc/deputy \
	--go-grpc_opt=Msrc/common.proto=github.com/open-cyber-range/vmware-handler/grpc/common \
	--go-grpc_opt=Msrc/template.proto=github.com/open-cyber-range/vmware-handler/grpc/template \
	--go-grpc_opt=Msrc/capability.proto=github.com/open-cyber-range/vmware-handler/grpc/capability  \
	--go-grpc_opt=Msrc/virtual-machine.proto=github.com/open-cyber-range/vmware-handler/grpc/virtual-machine  \
	--go-grpc_opt=Msrc/switch.proto=github.com/open-cyber-range/vmware-handler/grpc/switch  \
	--go-grpc_opt=Msrc/feature.proto=github.com/open-cyber-range/vmware-handler/grpc/feature  \
	--go-grpc_opt=Msrc/condition.proto=github.com/open-cyber-range/vmware-handler/grpc/condition  \
	--go-grpc_opt=Msrc/inject.proto=github.com/open-cyber-range/vmware-handler/grpc/inject  \
	--go-grpc_opt=Msrc/event.proto=github.com/open-cyber-range/vmware-handler/grpc/event  \
	--go-grpc_opt=Msrc/deputy.proto=github.com/open-cyber-range/vmware-handler/grpc/deputy  \
	--go_opt=module=github.com/open-cyber-range/vmware-handler/grpc \
	--go-grpc_opt=module=github.com/open-cyber-range/vmware-handler/grpc \
	--proto_path=grpc/proto src/virtual-machine.proto src/switch.proto src/common.proto src/capability.proto src/template.proto src/feature.proto src/condition.proto src/inject.proto src/event.proto src/deputy.proto

generate-nsx-t-openapi:
	java -Dapis=Segments,Connectivity -Dmodels -DsupportingFiles -jar /var/opt/swagger/swagger-codegen-cli.jar generate -DpackageName=nsx_t_openapi -DmodelTests=false -DapiTests=false -DapiDocs=false -DmodelDocs=false -D io.swagger.parser.util.RemoteUrl.trustAll=true -i extra/nsx_policy_api.yaml -l go -o nsx_t_openapi &&\
	find ./nsx_t_openapi -type f ! \( -name '*.go' \) -exec rm -f {} +

build: compile-protobuf build-machiner build-switcher build-templater build-executor build-general

build-machiner: compile-protobuf
	go build -o bin/ranger-vmware-machiner ./machiner

build-switcher: compile-protobuf
	go build -o bin/ranger-vmware-switcher ./switcher

build-templater: compile-protobuf
	go build -o bin/ranger-vmware-templater ./templater

build-executor: compile-protobuf
	go build -o bin/ranger-vmware-executor ./executor

build-general: compile-protobuf
	go build -o bin/ranger-vmware-general ./general

test: build test-machiner test-switcher test-templater test-executor test-general

test-machiner: build-machiner
	go test -v -count=1 -timeout 30m ./machiner

test-switcher: build-switcher
	go test -v -count=1 -timeout 30m ./switcher

test-templater: build-templater
	go test -v -count=1 -timeout 30m ./templater

test-executor: build-executor
	go test -v -count=1 -timeout 30m ./executor
	
test-injects: build-executor
	go test -v -count=1 ./executor -run TestInjectDeploymentAndDeletionOnLinux

test-permissions: build-executor
	go test -v -count=1 ./executor -run TestFeatureFilePermissionsOnLinux

test-windows: build-executor
	go test -v -count=1 ./executor -run "(TestConditionerWithSourcePackageOnWindows|TestFeatureServiceDeploymentAndDeletionOnWindows|TestRebootingFeatureOnWindows|TestConditionRebootWhileStreamingOnWindows)"

test-environment: build-executor
	go test -v -count=1 ./executor -run "(TestFeatureServiceDeploymentWithEnvironmentOnLinux|TestConditionerWithCommandAndEnvironment)"

test-general: build-general
	go test -v -count=1 ./general

test-deputy: build-general
	go test -v -count=1 ./general -run "(TestGetDeputyPackagesByType|TestGetScenario)"

test-library:
	go test -v -count=1 ./library

run-machiner: build-machiner
	bin/ranger-vmware-machiner machiner-config.yml

run-switcher: build-switcher
	bin/ranger-vmware-switcher switcher-config.yml

run-templater: build-templater
	bin/ranger-vmware-templater

run-executor: build-executor
	bin/ranger-vmware-executor

run-general: build-general
	bin/ranger-vmware-general

test-and-build: test build

uninstall:
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-machiner
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-switcher
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-templater
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-executor
	-rm -f $(DESTDIR)/var/opt/ranger/bin/ranger-vmware-general
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-machiner/config.yml
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-switcher/config.yml
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-templater/config.yml
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-executor/config.yml
	-rm -f $(DESTDIR)/etc/opt/ranger/ranger-vmware-general/config.yml
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-machiner.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-machiner.service
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-switcher.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-switcher.service
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-templater.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-templater.service
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-executor.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-executor.service
	-rm -f $(DESTDIR)/etc/systemd/system/ranger-vmware-general.service
	-rm -f $(DESTDIR)/lib/systemd/system/ranger-vmware-general.service

build-deb: clean build
	env DH_VERBOSE=1 dpkg-buildpackage -b --no-sign
