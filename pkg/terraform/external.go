package terraform

import (
	"context"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	xpmeta "github.com/crossplane/crossplane-runtime/pkg/meta"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	xpresource "github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane-contrib/terrajet/pkg/meta"
	"github.com/crossplane-contrib/terrajet/pkg/terraform/resource"
)

const (
	errUnexpectedObject = "The managed resource is not an Terraformed resource"
)

// An ExternalOption configures an External.
type ExternalOption func(*External)

// WithLogger specifies how the Reconciler should log messages.
func WithLogger(l logging.Logger) ExternalOption {
	return func(e *External) {
		e.log = l
	}
}

// WithRecorder specifies how the Reconciler should record events.
func WithRecorder(er event.Recorder) ExternalOption {
	return func(e *External) {
		e.record = er
	}
}

// NewExternal returns a terraform external client
func NewExternal(client client.Client, mg xpresource.Managed, providerConfig []byte, o ...ExternalOption) (*External, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return nil, errors.New(errUnexpectedObject)
	}

	e := &External{
		client: client,
		log:    logging.NewNopLogger(),
		record: event.NewNopRecorder(),
	}

	for _, eo := range o {
		eo(e)
	}

	e.tfClient = NewClient(e.log, providerConfig, tr)

	return e, nil
}

// External manages lifecycle of a Terraform managed resource by implementing
// managed.ExternalClient interface.
type External struct {
	client   client.Client
	tfClient Adapter

	log    logging.Logger
	record event.Recorder
}

// Observe does an observation for the Terraform managed resource.
func (e *External) Observe(ctx context.Context, mg xpresource.Managed) (managed.ExternalObservation, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errUnexpectedObject)
	}

	if xpmeta.GetExternalName(tr) == "" {
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	res, err := e.tfClient.Observe(ctx, tr)
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
func (e *External) Create(ctx context.Context, mg xpresource.Managed) (managed.ExternalCreation, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errUnexpectedObject)
	}

	res, err := e.tfClient.Create(ctx, tr)
	if err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "failed to create")
	}
	if !res.Completed {
		// Creation is in progress, do nothing
		return managed.ExternalCreation{}, nil
	}

	if err := e.persistState(ctx, tr, res.State, res.ExternalName); err != nil {
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot persist state")
	}

	return managed.ExternalCreation{
		ConnectionDetails: res.ConnectionDetails,
	}, err
}

// Update updates the Terraform managed resource.
func (e *External) Update(ctx context.Context, mg xpresource.Managed) (managed.ExternalUpdate, error) {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errUnexpectedObject)
	}

	res, err := e.tfClient.Update(ctx, tr)
	if err != nil {
		return managed.ExternalUpdate{}, errors.Wrap(err, "failed to update")
	}
	if !res.Completed {
		// Update is in progress, do nothing
		return managed.ExternalUpdate{}, nil
	}

	if meta.GetState(tr) != res.State {
		if err := e.persistState(ctx, tr, res.State, ""); err != nil {
			return managed.ExternalUpdate{}, errors.Wrap(err, "cannot persist state")
		}
	}

	return managed.ExternalUpdate{
		ConnectionDetails: res.ConnectionDetails,
	}, nil
}

// Delete deletes the Terraform managed resource.
func (e *External) Delete(ctx context.Context, mg xpresource.Managed) error {
	tr, ok := mg.(resource.Terraformed)
	if !ok {
		return errors.New(errUnexpectedObject)
	}

	_, err := e.tfClient.Delete(ctx, tr)
	if err != nil {
		return errors.Wrap(err, "failed to delete")
	}

	return nil
}

func (e *External) persistState(ctx context.Context, tr resource.Terraformed, state, externalName string) error {
	// We will retry in all cases where the error comes from the api-server.
	// At one point, context deadline will be exceeded and we'll get out
	// of the loop. In that case, we warn the user that the external resource
	// might be leaked.
	return errors.Wrap(retry.OnError(retry.DefaultRetry, xpresource.IsAPIError, func() error {
		nn := types.NamespacedName{Name: tr.GetName()}
		if err := e.client.Get(ctx, nn, tr); err != nil {
			return err
		}
		if xpmeta.GetExternalName(tr) == "" {
			xpmeta.SetExternalName(tr, externalName)
		}
		meta.SetState(tr, state)
		return e.client.Update(ctx, tr)
	}), "cannot update resource state")
}
