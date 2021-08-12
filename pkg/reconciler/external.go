package reconciler

import (
	"context"

	"github.com/pkg/errors"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/terrajet/pkg/adapter"
	"github.com/crossplane-contrib/terrajet/pkg/resource"
)

const (
	errUnexpectedObject = "The managed resource is not an Terraformed resource"
)

// NewTerraformExternal returns a terraform external client
func NewTerraformExternal(l logging.Logger, providerConfig []byte, mg xpresource.Managed) (*TerraformExternal, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return nil, errors.New(errUnexpectedObject)
	}

	return &TerraformExternal{
		adapter: adapter.NewTerraformCli(l, providerConfig, tr),
	}, nil
}

// TerraformExternal manages lifecycle of a Terraform managed resource by implementing
// managed.ExternalClient interface.
type TerraformExternal struct {
	adapter adapter.Adapter
}

// Observe does an observation for the Terraform managed resource.
func (e *TerraformExternal) Observe(ctx context.Context, mg xpresource.Managed) (managed.ExternalObservation, error) {
	_, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	return managed.ExternalObservation{
		ResourceExists: true,
	}, nil
}

// Create creates the Terraform managed resource.
func (e *TerraformExternal) Create(ctx context.Context, mg xpresource.Managed) (managed.ExternalCreation, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	cr, err := e.adapter.Create(ctx, tr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create")
	}
	if !cr.Completed {
		return managed.ExternalCreation{}, errors.Wrap(err, "create in progress")
	}

	return managed.ExternalCreation{
		ExternalNameAssigned: meta.GetExternalName(mg) != "",
		ConnectionDetails:    cr.ConnectionDetails,
	}, err
}

// Update updates the Terraform managed resource.
func (e *TerraformExternal) Update(ctx context.Context, mg xpresource.Managed) (managed.ExternalUpdate, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}

	cr, err := e.adapter.Update(ctx, tr)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update")
	}
	if !cr.Completed {
		return managed.ExternalUpdate{}, errors.Wrap(err, "update in progress")
	}

	return managed.ExternalUpdate{
		ConnectionDetails: cr.ConnectionDetails,
	}, nil
}

// Delete deletes the Terraform managed resource.
func (e *TerraformExternal) Delete(ctx context.Context, mg xpresource.Managed) error {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	cr, err := e.adapter.Delete(ctx, tr)
	if err != nil {
		return errors.Wrap(err, "failed to update")
	}
	if !cr.Completed {
		return errors.Wrap(err, "update in progress")
	}

	return errors.Errorf("still deleting")
}
