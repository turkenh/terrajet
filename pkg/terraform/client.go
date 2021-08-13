package terraform

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"

	"github.com/crossplane-contrib/terrajet/pkg/meta"
	"github.com/crossplane-contrib/terrajet/pkg/terraform/resource"
	"github.com/crossplane-contrib/terrajet/pkg/tfcli"
)

// Client is an Adapter implementation for Terraform Cli
type Client struct {
	builderBase tfcli.Builder
}

func NewClient(l logging.Logger, providerConfig []byte, tr resource.Terraformed) *Client {
	tfcb := tfcli.NewClientBuilder().
		WithLogger(l).
		WithResourceName(tr.GetName()).
		WithHandle(string(tr.GetUID())).
		WithProviderConfiguration(providerConfig).
		WithResourceType(tr.GetTerraformResourceType())

	return &Client{
		builderBase: tfcb,
	}
}

func (t *Client) Observe(ctx context.Context, tr resource.Terraformed) (ObserveResult, error) {
	// TODO(hasan): Need to get refreshed state once cli interface has that functionality
	stEnc := meta.GetState(tr)
	st, err := BuildStateV4(stEnc, nil)
	if err != nil {
		return ObserveResult{}, errors.Wrap(err, "cannot build state")
	}

	if err = tr.SetParameters(st.GetAttributes()); err != nil {
		return ObserveResult{}, errors.Wrap(err, "cannot set parameters")
	}

	if err = tr.SetObservation(st.GetAttributes()); err != nil {
		return ObserveResult{}, errors.Wrap(err, "cannot set parameters")
	}

	return ObserveResult{
		Completed:         true,
		State:             "",
		ConnectionDetails: nil,
		UpToDate:          true,
		Exists:            true,
		LateInitialized:   false,
	}, nil
}

func (t *Client) Create(ctx context.Context, tr resource.Terraformed) (CreateResult, error) {
	res := CreateResult{}

	attr, err := tr.GetParameters()
	if err != nil {
		return res, errors.Wrap(err, "failed to get attributes")
	}

	tfc, err := t.builderBase.WithResourceBody(attr).BuildCreateClient()

	if err != nil {
		return res, errors.Wrap(err, "cannot build create client")
	}

	completed, err := tfc.Create()
	if err != nil {
		return res, errors.Wrap(err, "create failed with")
	}

	if !completed {
		return res, nil
	}
	res.Completed = true

	stRaw := tfc.GetState()
	st, err := ReadStateV4(stRaw)
	if err != nil {
		return res, errors.Wrap(err, "cannot parse state")
	}

	stAttr := map[string]interface{}{}

	if err = json.Unmarshal(st.GetAttributes(), &stAttr); err != nil {
		return res, errors.Wrap(err, "cannot parse state attributes")
	}

	id, exists := stAttr[tr.GetTerraformResourceIdField()]
	if !exists {
		return res, errors.Wrap(err, fmt.Sprintf("no value for id field: %s", tr.GetTerraformResourceIdField()))
	}
	en, ok := id.(string)
	if !ok {
		return res, errors.Wrap(err, "id field is not a string")
	}
	res.ExternalName = en

	if res.State, err = st.GetEncodedState(); err != nil {
		return res, errors.Wrap(err, "cannot get encoded state")
	}

	conn := managed.ConnectionDetails{}
	sensitive := st.GetSensitiveAttributes()
	if sensitive != nil {
		if err = json.Unmarshal(sensitive, &conn); err != nil {
			return res, errors.Wrap(err, "cannot parse connection details")
		}
	}
	res.ConnectionDetails = conn

	return res, nil
}

// Update is a Terraform Cli implementation for Apply function of Adapter interface.
func (t *Client) Update(ctx context.Context, tr resource.Terraformed) (UpdateResult, error) {
	return UpdateResult{}, nil
}

// Delete is a Terraform Cli implementation for Delete function of Adapter interface.
func (t *Client) Delete(ctx context.Context, tr resource.Terraformed) (DeletionResult, error) {
	res := DeletionResult{}

	stEnc := meta.GetState(tr)
	st, err := BuildStateV4(stEnc, nil)
	if err != nil {
		return res, errors.Wrap(err, "cannot build state")
	}

	stRaw, err := st.Serialize()
	if err != nil {
		return res, errors.Wrap(err, "cannot serialize state")
	}

	attr, err := tr.GetParameters()
	if err != nil {
		return res, errors.Wrap(err, "failed to get attributes")
	}

	tfc, err := t.builderBase.WithState(stRaw).WithResourceBody(attr).BuildDeletionClient()
	if err != nil {
		return res, errors.Wrap(err, "cannot build delete client")
	}

	completed, err := tfc.Delete()
	if err != nil {
		return res, errors.Wrap(err, "failed to delete")
	}
	if !completed {
		return res, nil
	}
	res.Completed = true
	return res, nil
}
