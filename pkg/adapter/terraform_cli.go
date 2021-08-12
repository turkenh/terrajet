package adapter

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/crossplane-contrib/terrajet/pkg/tfcli"
	"github.com/crossplane/crossplane-runtime/pkg/logging"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"

	"github.com/crossplane-contrib/terrajet/pkg/resource"
)

// TerraformCli is an Adapter implementation for Terraform Cli
type TerraformCli struct {
	builderBase tfcli.Builder
}

func NewTerraformCli(l logging.Logger, providerConfig []byte, tr resource.Terraformed) *TerraformCli {
	tfcb := tfcli.NewClientBuilder().
		WithLogger(l).
		WithResourceName(tr.GetName()).
		WithHandle(string(tr.GetUID())).
		WithProviderConfiguration(providerConfig).
		WithResourceType(tr.GetTerraformResourceType())

	return &TerraformCli{
		builderBase: tfcb,
	}
}

// Exists is a Terraform Cli implementation for Exists function of Adapter interface.
func (t *TerraformCli) Exists(ctx context.Context, tr resource.Terraformed) (bool, error) {
	fmt.Println("Terraform Cli adapter checking if exists...")
	return true, nil
}

// UpdateStatus is a Terraform Cli implementation for UpdateStatus function of Adapter interface.
func (t *TerraformCli) UpdateStatus(ctx context.Context, tr resource.Terraformed) error {
	/*if err := tr.SetObservation([]byte{}); err != nil {
		return errors.Wrap(err, "failed to set observation")
	}*/
	fmt.Println("Terraform Cli adapter updating status...")
	return nil
}

// LateInitialize is a Terraform Cli implementation for LateInitialize function of Adapter interface.
func (t *TerraformCli) LateInitialize(ctx context.Context, tr resource.Terraformed) (bool, error) {
	/*if err := tr.SetParameters([]byte{}); err != nil {
		return false, errors.Wrap(err, "failed to set parameters")
	}*/
	fmt.Println("Terraform Cli adapter late initializing...")
	return true, nil
}

// IsReady is a Terraform Cli implementation for IsReady function of Adapter interface.
func (t *TerraformCli) IsReady(ctx context.Context, tr resource.Terraformed) (bool, error) {
	fmt.Println("Terraform Cli adapter checking if ready...")
	return true, nil
}

// IsUpToDate is a Terraform Cli implementation for IsUpToDate function of Adapter interface.
func (t *TerraformCli) IsUpToDate(ctx context.Context, tr resource.Terraformed) (bool, error) {
	fmt.Println("Terraform Cli adapter checking if up to date...")
	return false, nil
}

// GetConnectionDetails is a Terraform Cli implementation for GetConnectionDetails function of Adapter interface.
func (t *TerraformCli) GetConnectionDetails(ctx context.Context, tr resource.Terraformed) (managed.ConnectionDetails, error) {
	fmt.Println("Terraform Cli adapter returning connection details...")
	return managed.ConnectionDetails{}, nil
}

func (t *TerraformCli) Create(ctx context.Context, tr resource.Terraformed) (CreateResult, error) {
	attr, err := tr.GetAttributes()
	if err != nil {
		return CreateResult{}, errors.Wrap(err, "failed to get attributes")
	}

	tfc, err := t.builderBase.
		WithResourceBody(attr).
		BuildCreateClient()

	if err != nil {
		return CreateResult{}, errors.Wrap(err, "cannot build create client")
	}

	completed, err := tfc.Create()
	if err != nil {
		return CreateResult{}, errors.Wrap(err, "create failed with")
	}

	if !completed {
		return CreateResult{}, errors.New("create in progress")
	}

	st := tfc.GetState()
	conn, err := tr.ConsumeState(st)
	if err != nil {
		return CreateResult{}, errors.Wrap(err, "failed to consume state")
	}

	return CreateResult{
		Completed:         true,
		ConnectionDetails: conn,
	}, nil
}

// Update is a Terraform Cli implementation for Apply function of Adapter interface.
func (t *TerraformCli) Update(ctx context.Context, tr resource.Terraformed) (UpdateResult, error) {
	return UpdateResult{}, nil
}

// Delete is a Terraform Cli implementation for Delete function of Adapter interface.
func (t *TerraformCli) Delete(ctx context.Context, tr resource.Terraformed) (DeletionResult, error) {
	tfcb := t.builderBase.
		WithResourceBody([]byte(`{ 
                "cidr_block": "10.0.0.0/16",
  
                "tags": {
                    "Name": "alper-example-terraform"
                }
           }`))

	tfc, err := tfcb.BuildDeletionClient()
	if err != nil {
		return DeletionResult{}, errors.Wrap(err, "cannot build create client")
	}

	completed, err := tfc.Delete()
	if err != nil {
		return DeletionResult{}, errors.Wrap(err, "failed to delete")
	}
	if completed {
		return DeletionResult{}, nil
	}
	return DeletionResult{Completed: true}, nil
}
