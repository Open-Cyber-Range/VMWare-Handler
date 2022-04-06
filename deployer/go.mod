module github.com/open-cyber-range/vmware-node-deployer/deployer

replace github.com/open-cyber-range/sdl-parser => ../../../../../../projects/ocr/sdl-parser/go-package

require (
	github.com/vmware/govmomi v0.27.4
	gopkg.in/yaml.v2 v2.4.0
)

go 1.17
