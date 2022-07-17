package main

import (
	"bytes"
	"io/ioutil"
	"log"

	"github.com/vmware/govmomi/govc/importx"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25"
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

func (templateDeployment *TemplateDeployment) ImportOVA(filePath string, client *vim25.Client) (err error) {
	ovfBytes, err := templateDeployment.readOvf(filePath, client)
	if err != nil {
		return
	}
	envelope, err := templateDeployment.readEnvelope(ovfBytes)
	if err != nil {
		return
	}
	log.Printf("Envelope %+v", envelope)

	return
}
