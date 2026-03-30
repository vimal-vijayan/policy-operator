package assignments

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
)

// Client wraps armpolicy.AssignmentsClient and implements the API interface.
type Client struct {
	inner *armpolicy.AssignmentsClient
}

// NewClient creates a new Client from an armpolicy.AssignmentsClient.
func NewClient(c *armpolicy.AssignmentsClient) *Client {
	return &Client{inner: c}
}

func (c *Client) Create(ctx context.Context, scope string, policyAssignmentName string, parameters armpolicy.Assignment, options *armpolicy.AssignmentsClientCreateOptions) (armpolicy.AssignmentsClientCreateResponse, error) {
	return c.inner.Create(ctx, scope, policyAssignmentName, parameters, options)
}

func (c *Client) Delete(ctx context.Context, scope string, policyAssignmentName string, options *armpolicy.AssignmentsClientDeleteOptions) (armpolicy.AssignmentsClientDeleteResponse, error) {
	return c.inner.Delete(ctx, scope, policyAssignmentName, options)
}

func (c *Client) Get(ctx context.Context, scope string, policyAssignmentName string, options *armpolicy.AssignmentsClientGetOptions) (armpolicy.AssignmentsClientGetResponse, error) {
	return c.inner.Get(ctx, scope, policyAssignmentName, options)
}

func (c *Client) GetByID(ctx context.Context, policyAssignmentID string, options *armpolicy.AssignmentsClientGetByIDOptions) (armpolicy.AssignmentsClientGetByIDResponse, error) {
	return c.inner.GetByID(ctx, policyAssignmentID, options)
}
