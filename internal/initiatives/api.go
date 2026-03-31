// Package initiatives contains the controller logic for AzurePolicyInitiative resources.
package initiatives

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
)

type API interface {
	CreateOrUpdate(ctx context.Context, policySetDefinitionName string, parameters armpolicy.SetDefinition, options *armpolicy.SetDefinitionsClientCreateOrUpdateOptions) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error)
	Delete(ctx context.Context, policySetDefinitionName string, options *armpolicy.SetDefinitionsClientDeleteOptions) (armpolicy.SetDefinitionsClientDeleteResponse, error)
	Get(ctx context.Context, policySetDefinitionName string, options *armpolicy.SetDefinitionsClientGetOptions) (armpolicy.SetDefinitionsClientGetResponse, error)
	CreateOrUpdateAtManagementGroup(ctx context.Context, managementGroupID string, policySetDefinitionName string, parameters armpolicy.SetDefinition, options *armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupOptions) (armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse, error)
	DeleteAtManagementGroup(ctx context.Context, managementGroupID string, policySetDefinitionName string, options *armpolicy.SetDefinitionsClientDeleteAtManagementGroupOptions) (armpolicy.SetDefinitionsClientDeleteAtManagementGroupResponse, error)
	GetAtManagementGroup(ctx context.Context, managementGroupID string, policySetDefinitionName string, options *armpolicy.SetDefinitionsClientGetAtManagementGroupOptions) (armpolicy.SetDefinitionsClientGetAtManagementGroupResponse, error)
}
