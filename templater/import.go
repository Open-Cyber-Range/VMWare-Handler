package main

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/vmware/govmomi/govc/importx"
	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type Archive struct {
	importx.TapeArchive
}

type Envelope struct {
	XMLName        xml.Name       `xml:"Envelope"`
	NetworkSection NetworkSection `xml:"NetworkSection"`
}

type NetworkSection struct {
	Info    string    `xml:"Info"`
	Network []Network `xml:"Network"`
}

type Network struct {
	Name        string `xml:"name,attr"`
	Description string `xml:"Description"`
}

func parseOVF(ovfBytes []byte) (*Envelope, error) {
	var envelope Envelope
	err := xml.Unmarshal(ovfBytes, &envelope)
	if err != nil {
		return nil, err
	}
	return &envelope, nil
}

func (templateDeployment *TemplateDeployment) readOvf(ovaArchive *Archive) (ovfBytes []byte, err error) {
	reader, _, err := ovaArchive.Open("*.ovf")
	if err != nil {
		return
	}
	defer reader.Close()

	ovfBytes, err = io.ReadAll(reader)
	return
}

func (templateDeployment *TemplateDeployment) ImportOVA(filePath string, client *vim25.Client) (importObject *types.ManagedObjectReference, err error) {
	ctx := context.Background()
	ovaArchive := Archive{
		TapeArchive: importx.TapeArchive{
			Path: filePath,
			Opener: importx.Opener{
				Client: client,
			},
		},
	}
	ovfBytes, err := templateDeployment.readOvf(&ovaArchive)
	if err != nil {
		return
	}

	ovfEnvelope, err := parseOVF(ovfBytes)
	if err != nil {
		return
	}

	ovfHasNetworks := len(ovfEnvelope.NetworkSection.Network) > 0

	var hostNetworkSystem *object.HostNetworkSystem
	if ovfHasNetworks {
		log.Infof("OVF has networks, creating temporary network infrastructure")

		hostSystem, err := templateDeployment.Client.PickHostSystem(ctx)
		if err != nil {
			log.Errorf("error picking host system: %v", err)
			return nil, err
		}

		hostNetworkSystem, err = hostSystem.ConfigManager().NetworkSystem(ctx)
		if err != nil {
			log.Errorf("error getting host network system: %v", err)
			return nil, err
		}

		if err = templateDeployment.Client.CreateTmpNetwork(ctx, hostSystem, hostNetworkSystem); err != nil {
			log.Errorf("error creating network infrastructure: %v", err)
			return nil, err
		}
	}

	cisp := types.OvfCreateImportSpecParams{
		EntityName: templateDeployment.templateName,
	}

	uploadManager := ovf.NewManager(templateDeployment.Client.Client.Client)
	resourcePool, err := templateDeployment.Client.GetResourcePool(templateDeployment.Configuration.ResourcePoolPath)
	if err != nil {
		log.Errorf("Failed to get resource pool: %v", err)
		return
	}
	datastore, err := templateDeployment.Client.GetDatastore(templateDeployment.Configuration.DatastorePath)
	if err != nil {
		log.Errorf("Failed to get datastore: %v", err)
		return
	}

	importSpec, err := uploadManager.CreateImportSpec(ctx, string(ovfBytes), resourcePool, datastore, cisp)
	if err != nil {
		log.Errorf("Failed to create import spec: %v", err)
		return
	}
	folder, err := templateDeployment.Client.GetTemplateFolder()
	if err != nil {
		return
	}
	lease, err := resourcePool.ImportVApp(ctx, importSpec.ImportSpec, folder, nil)
	if err != nil {
		log.Errorf("Failed to import OVA: %v", err)
		return
	}

	info, err := lease.Wait(ctx, importSpec.FileItem)
	if err != nil {
		return
	}

	updater := lease.StartUpdater(ctx, info)
	defer updater.Done()

	for _, i := range info.Items {
		ovaArchive.Upload(ctx, lease, i)
		if err != nil {
			return
		}
	}

	if ovfHasNetworks {
		if err := templateDeployment.Client.CleanupTmpNetwork(ctx, hostNetworkSystem); err != nil {
			log.Errorf("error cleaning up tmp ovf infrastructure: %v", err)
			return nil, err
		}
	}

	return &info.Entity, lease.Complete(ctx)
}

func (archive *Archive) Upload(ctx context.Context, lease *nfc.Lease, item nfc.FileItem) error {
	file := item.Path

	f, size, err := archive.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()
	logger := NewProgressLogger(fmt.Sprintf("Uploading %s... ", path.Base(file)))

	opts := soap.Upload{
		ContentLength: size,
		Progress:      logger,
	}

	return lease.Upload(ctx, item, f, opts)
}
