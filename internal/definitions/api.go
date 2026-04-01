// Package definitions contains the controller logic for AzurePolicyDefinition resources.
package definitions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
)

type API interface {
	CreateOrUpdate(ctx context.Context, policyDefinitionName string, parameters armpolicy.Definition, options *armpolicy.DefinitionsClientCreateOrUpdateOptions) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error)
	Delete(ctx context.Context, policyDefinitionName string, options *armpolicy.DefinitionsClientDeleteOptions) (armpolicy.DefinitionsClientDeleteResponse, error)
	Get(ctx context.Context, policyDefinitionName string, options *armpolicy.DefinitionsClientGetOptions) (armpolicy.DefinitionsClientGetResponse, error)
	CreateOrUpdateAtManagementGroup(ctx context.Context, managementGroupID string, policyDefinitionName string, parameters armpolicy.Definition, options *armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupOptions) (armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse, error)
	DeleteAtManagementGroup(ctx context.Context, managementGroupID string, policyDefinitionName string, options *armpolicy.DefinitionsClientDeleteAtManagementGroupOptions) (armpolicy.DefinitionsClientDeleteAtManagementGroupResponse, error)
	GetAtManagementGroup(ctx context.Context, managementGroupID string, policyDefinitionName string, options *armpolicy.DefinitionsClientGetAtManagementGroupOptions) (armpolicy.DefinitionsClientGetAtManagementGroupResponse, error)
}
