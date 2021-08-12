package adapter

import (
	"context"

	"github.com/crossplane-contrib/terrajet/pkg/meta"

	"github.com/crossplane-contrib/terrajet/pkg/conversion"
	"k8s.io/apimachinery/pkg/util/json"

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

func (t *TerraformCli) Observe(ctx context.Context, tr resource.Terraformed) (ObserveResult, error) {
	// TODO(hasan): Need to get refreshed state once cli interface has that functionality
	stEnc := meta.GetState(tr)
	st, err := conversion.BuildStateV4(stEnc, nil)
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

func (t *TerraformCli) Create(ctx context.Context, tr resource.Terraformed) (CreateResult, error) {
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
	st, err := conversion.ReadStateV4(stRaw)
	if err != nil {
		return res, errors.Wrap(err, "cannot parse state")
	}

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
func (t *TerraformCli) Update(ctx context.Context, tr resource.Terraformed) (UpdateResult, error) {
	return UpdateResult{}, nil
}

// Delete is a Terraform Cli implementation for Delete function of Adapter interface.
func (t *TerraformCli) Delete(ctx context.Context, tr resource.Terraformed) (DeletionResult, error) {
	res := DeletionResult{}

	stEnc := meta.GetState(tr)
	st, err := conversion.BuildStateV4(stEnc, nil)
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
