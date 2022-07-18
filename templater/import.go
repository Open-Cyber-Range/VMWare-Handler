package main

import (
	"bytes"
	"context"
	"io/ioutil"
	"log"

	"github.com/vmware/govmomi/govc/importx"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/types"
)

func (templateDeployment *TemplateDeployment) readOvf(filePath string, client *vim25.Client) (ovfBytes []byte, err error) {
	ovaArchive := importx.TapeArchive{
		Path: filePath,
		Opener: importx.Opener{
			Client: client,
		},
	}
	reader, _, err := ovaArchive.Open("*.ovf")
	if err != nil {
		return
	}
	defer reader.Close()

	ovfBytes, err = ioutil.ReadAll(reader)
	return
}

func (templateDeployment *TemplateDeployment) readEnvelope(ovfBytes []byte) (envelope *ovf.Envelope, err error) {
	envelope, err = ovf.Unmarshal(bytes.NewReader(ovfBytes))
	return
}

func (templateDeployment *TemplateDeployment) ImportOVA(filePath string, client *vim25.Client, cheksum string) (err error) {
	ovfBytes, err := templateDeployment.readOvf(filePath, client)
	if err != nil {
		return
	}
	envelope, err := templateDeployment.readEnvelope(ovfBytes)
	if err != nil {
		return
	}
	log.Printf("Envelope %+v", envelope)

	cisp := types.OvfCreateImportSpecParams{
		EntityName: cheksum,
	}
	uploadManager := ovf.NewManager(templateDeployment.Client.Client.Client)
	ctx := context.Background()

	resourcePool, err := templateDeployment.Client.GetResourcePool(templateDeployment.Configuration.ResourcePoolPath)
	if err != nil {
		return
	}
	datastore, err := templateDeployment.Client.GetDatastore("/Datacenter/datastore/datastore2")
	if err != nil {
		return
	}

	importSpec, err := uploadManager.CreateImportSpec(ctx, string(ovfBytes), resourcePool, datastore, cisp)
	if err != nil {
		return
	}
	log.Printf("ImportSpec %+v", importSpec)
	folder, err := templateDeployment.Client.GetTemplateFolder()
	if err != nil {
		return
	}
	lease, err := resourcePool.ImportVApp(ctx, importSpec.ImportSpec, folder, nil)
	if err != nil {
		return
	}
	log.Printf("Lease %+v", lease)

	return
}
