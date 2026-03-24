package exemptions

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
)

// Client wraps armpolicy.ExemptionsClient and implements the API interface.
type Client struct {
	inner *armpolicy.ExemptionsClient
}

// NewClient creates a new Client from an armpolicy.ExemptionsClient.
func NewClient(c *armpolicy.ExemptionsClient) *Client {
	return &Client{inner: c}
}

func (c *Client) CreateOrUpdate(ctx context.Context, scope string, policyExemptionName string, parameters armpolicy.Exemption, options *armpolicy.ExemptionsClientCreateOrUpdateOptions) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
	return c.inner.CreateOrUpdate(ctx, scope, policyExemptionName, parameters, options)
}

func (c *Client) Delete(ctx context.Context, scope string, policyExemptionName string, options *armpolicy.ExemptionsClientDeleteOptions) (armpolicy.ExemptionsClientDeleteResponse, error) {
	return c.inner.Delete(ctx, scope, policyExemptionName, options)
}

func (c *Client) Get(ctx context.Context, scope string, policyExemptionName string, options *armpolicy.ExemptionsClientGetOptions) (armpolicy.ExemptionsClientGetResponse, error) {
	return c.inner.Get(ctx, scope, policyExemptionName, options)
}
