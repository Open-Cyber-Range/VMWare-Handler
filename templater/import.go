package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"

	"github.com/vmware/govmomi/govc/importx"
	"github.com/vmware/govmomi/nfc"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type Archive struct {
	importx.TapeArchive
}

func (templateDeployment *TemplateDeployment) readOvf(ovaArchive *Archive) (ovfBytes []byte, err error) {
	reader, _, err := ovaArchive.Open("*.ovf")
	if err != nil {
		return
	}
	defer reader.Close()

	ovfBytes, err = ioutil.ReadAll(reader)
	return
}

func (templateDeployment *TemplateDeployment) ImportOVA(filePath string, client *vim25.Client) (importObject *types.ManagedObjectReference, err error) {
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

	cisp := types.OvfCreateImportSpecParams{
		EntityName: templateDeployment.templateName,
	}
	uploadManager := ovf.NewManager(templateDeployment.Client.Client.Client)
	ctx := context.Background()

	resourcePool, err := templateDeployment.Client.GetResourcePool(templateDeployment.Configuration.ResourcePoolPath)
	if err != nil {
		return
	}
	datastore, err := templateDeployment.Client.GetDatastore(templateDeployment.Configuration.DatastorePath)
	if err != nil {
		return
	}

	importSpec, err := uploadManager.CreateImportSpec(ctx, string(ovfBytes), resourcePool, datastore, cisp)
	if err != nil {
		return
	}
	folder, err := templateDeployment.Client.GetTemplateFolder()
	if err != nil {
		return
	}
	lease, err := resourcePool.ImportVApp(ctx, importSpec.ImportSpec, folder, nil)
	if err != nil {
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
