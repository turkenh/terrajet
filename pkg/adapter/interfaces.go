package adapter

import (
	"context"

	"github.com/crossplane-contrib/terrajet/pkg/resource"

	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
)

// CreateResult represents result of a create operation
type CreateResult struct {
	// Tells whether the apply operation is completed.
	Completed bool

	// Sensitive information that is available during creation/update.
	ConnectionDetails managed.ConnectionDetails
}

// UpdateResult represents result of an update operation
type UpdateResult struct {
	// Tells whether the apply operation is completed.
	Completed bool

	// Sensitive information that is available during creation/update.
	ConnectionDetails managed.ConnectionDetails
}

// DeletionResult represents result of a delete operation
type DeletionResult struct {
	// Tells whether the apply operation is completed.
	Completed bool
}

// A Adapter is used to interact with terraform managed resources
type Adapter interface {
	Exists(ctx context.Context, tr resource.Terraformed) (bool, error)
	UpdateStatus(ctx context.Context, tr resource.Terraformed) error
	LateInitialize(ctx context.Context, tr resource.Terraformed) (bool, error)
	IsReady(ctx context.Context, tr resource.Terraformed) (bool, error)
	IsUpToDate(ctx context.Context, tr resource.Terraformed) (bool, error)
	GetConnectionDetails(ctx context.Context, tr resource.Terraformed) (managed.ConnectionDetails, error)
	Create(ctx context.Context, tr resource.Terraformed) (CreateResult, error)
	Update(ctx context.Context, tr resource.Terraformed) (UpdateResult, error)
	Delete(ctx context.Context, tr resource.Terraformed) (DeletionResult, error)
}
