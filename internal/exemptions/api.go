// Package exemptions contains the controller logic for AzurePolicyExemption resources.
package exemptions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
)

type API interface {
	CreateOrUpdate(ctx context.Context, scope string, policyExemptionName string, parameters armpolicy.Exemption, options *armpolicy.ExemptionsClientCreateOrUpdateOptions) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error)
	Delete(ctx context.Context, scope string, policyExemptionName string, options *armpolicy.ExemptionsClientDeleteOptions) (armpolicy.ExemptionsClientDeleteResponse, error)
	Get(ctx context.Context, scope string, policyExemptionName string, options *armpolicy.ExemptionsClientGetOptions) (armpolicy.ExemptionsClientGetResponse, error)
}
