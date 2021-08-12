package reconciler

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/terrajet/pkg/adapter"
	"github.com/crossplane-contrib/terrajet/pkg/meta"
	"github.com/crossplane-contrib/terrajet/pkg/resource"
)

const (
	errUnexpectedObject = "The managed resource is not an Terraformed resource"
)

// NewTerraformExternal returns a terraform external client
func NewTerraformExternal(client client.Client, l logging.Logger, providerConfig []byte, mg xpresource.Managed) (*TerraformExternal, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return nil, errors.New(errUnexpectedObject)
	}

	return &TerraformExternal{
		client:  client,
		adapter: adapter.NewTerraformCli(l, providerConfig, tr),
		log:     l,
	}, nil
}

// TerraformExternal manages lifecycle of a Terraform managed resource by implementing
// managed.ExternalClient interface.
type TerraformExternal struct {
	client  client.Client
	adapter adapter.Adapter

	log    logging.Logger
	record event.Recorder
}

// Observe does an observation for the Terraform managed resource.
func (e *TerraformExternal) Observe(ctx context.Context, mg xpresource.Managed) (managed.ExternalObservation, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	if meta.GetState(tr) == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	res, err := e.adapter.Observe(ctx, tr)
	if err != nil {
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot check if resource exists")
	}

	if !res.Completed {
		// Observation is in progress, do nothing
		tr.SetConditions(xpv1.Creating())
		return managed.ExternalObservation{
			ResourceExists:   true,
			ResourceUpToDate: true,
		}, nil
	}

	tr.SetConditions(xpv1.Available())

	return managed.ExternalObservation{
		ResourceExists:          res.Exists,
		ResourceUpToDate:        res.UpToDate,
		ResourceLateInitialized: res.LateInitialized,
		ConnectionDetails:       res.ConnectionDetails,
	}, nil
}

// Create creates the Terraform managed resource.
func (e *TerraformExternal) Create(ctx context.Context, mg xpresource.Managed) (managed.ExternalCreation, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	res, err := e.adapter.Create(ctx, tr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create")
	}
	if !res.Completed {
		// Creation is in progress, do nothing
		return managed.ExternalCreation{}, nil
	}

	if err := e.persistState(ctx, tr, res.State); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot persist state")
	}

	return managed.ExternalCreation{
		ConnectionDetails: res.ConnectionDetails,
	}, err
}

// Update updates the Terraform managed resource.
func (e *TerraformExternal) Update(ctx context.Context, mg xpresource.Managed) (managed.ExternalUpdate, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}

	res, err := e.adapter.Update(ctx, tr)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update")
	}
	if !res.Completed {
		// Update is in progress, do nothing
		return managed.ExternalUpdate{}, nil
	}

	if meta.GetState(tr) != res.State {
		if err := e.persistState(ctx, tr, res.State); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "cannot persist state")
		}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: res.ConnectionDetails,
	}, nil
}

// Delete deletes the Terraform managed resource.
func (e *TerraformExternal) Delete(ctx context.Context, mg xpresource.Managed) error {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	res, err := e.adapter.Delete(ctx, tr)
	if err != nil {
		return errors.Wrap(err, "failed to update")
	}
	if !res.Completed {
		return errors.Wrap(err, "update in progress")
	}

	// Deletion is in progress, do nothing
	return nil
	//return errors.Errorf("still deleting")
}

func (e *TerraformExternal) persistState(ctx context.Context, tr resource.Terraformed, state string) error {
	// We will retry in all cases where the error comes from the api-server.
	// At one point, context deadline will be exceeded and we'll get out
	// of the loop. In that case, we warn the user that the external resource
	// might be leaked.
	return errors.Wrap(retry.OnError(retry.DefaultRetry, xpresource.IsAPIError, func() error {
		nn := types.NamespacedName{Name: tr.GetName()}
		if err := e.client.Get(ctx, nn, tr); err != nil {
			return err
		}
		meta.SetState(tr, state)
		return e.client.Update(ctx, tr)
	}), "cannot update resource state")
}
