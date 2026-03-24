// Package assignments contains the controller logic for AzurePolicyAssignment resources.
package assignments

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
)

type API interface {
	Create(ctx context.Context, scope string, policyAssignmentName string, parameters armpolicy.Assignment, options *armpolicy.AssignmentsClientCreateOptions) (armpolicy.AssignmentsClientCreateResponse, error)
	Delete(ctx context.Context, scope string, policyAssignmentName string, options *armpolicy.AssignmentsClientDeleteOptions) (armpolicy.AssignmentsClientDeleteResponse, error)
	Get(ctx context.Context, scope string, policyAssignmentName string, options *armpolicy.AssignmentsClientGetOptions) (armpolicy.AssignmentsClientGetResponse, error)
}
