package resource

import (
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
)

// TerraformStateHandler handles terraform state
type TerraformStateHandler interface {
	GetState() ([]byte, error)
	ConsumeState(data []byte) (managed.ConnectionDetails, error)

	GetAttributes() ([]byte, error)
}

type TerraformMetadataProvider interface {
	GetTerraformResourceType() string
}

// Terraformed is a Kubernetes object representing a concrete terraform managed resource
type Terraformed interface {
	resource.Managed

	TerraformMetadataProvider
	TerraformStateHandler
}
