package client_test

import (
	"context"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/vimal-vijayan/azure-policy-operator/internal/assignments"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
	"github.com/vimal-vijayan/azure-policy-operator/internal/definitions"
	"github.com/vimal-vijayan/azure-policy-operator/internal/exemptions"
	"github.com/vimal-vijayan/azure-policy-operator/internal/initiatives"
)

// ── Test helpers ─────────────────────────────────────────────────────────────

// fakeDefinitionsAPI is a test double for definitions.API.
type fakeDefinitionsAPI struct{}

func (f *fakeDefinitionsAPI) CreateOrUpdate(_ context.Context, _ string, _ armpolicy.Definition, _ *armpolicy.DefinitionsClientCreateOrUpdateOptions) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
	return armpolicy.DefinitionsClientCreateOrUpdateResponse{}, nil
}

func (f *fakeDefinitionsAPI) Delete(_ context.Context, _ string, _ *armpolicy.DefinitionsClientDeleteOptions) (armpolicy.DefinitionsClientDeleteResponse, error) {
	return armpolicy.DefinitionsClientDeleteResponse{}, nil
}

func (f *fakeDefinitionsAPI) Get(_ context.Context, _ string, _ *armpolicy.DefinitionsClientGetOptions) (armpolicy.DefinitionsClientGetResponse, error) {
	return armpolicy.DefinitionsClientGetResponse{}, nil
}

func (f *fakeDefinitionsAPI) CreateOrUpdateAtManagementGroup(_ context.Context, _ string, _ string, _ armpolicy.Definition, _ *armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupOptions) (armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse, error) {
	return armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse{}, nil
}

func (f *fakeDefinitionsAPI) DeleteAtManagementGroup(_ context.Context, _ string, _ string, _ *armpolicy.DefinitionsClientDeleteAtManagementGroupOptions) (armpolicy.DefinitionsClientDeleteAtManagementGroupResponse, error) {
	return armpolicy.DefinitionsClientDeleteAtManagementGroupResponse{}, nil
}

func (f *fakeDefinitionsAPI) GetAtManagementGroup(_ context.Context, _ string, _ string, _ *armpolicy.DefinitionsClientGetAtManagementGroupOptions) (armpolicy.DefinitionsClientGetAtManagementGroupResponse, error) {
	return armpolicy.DefinitionsClientGetAtManagementGroupResponse{}, nil
}

// fakeInitiativesAPI is a test double for initiatives.API.
type fakeInitiativesAPI struct{}

func (f *fakeInitiativesAPI) CreateOrUpdate(_ context.Context, _ string, _ armpolicy.SetDefinition, _ *armpolicy.SetDefinitionsClientCreateOrUpdateOptions) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
	return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{}, nil
}

func (f *fakeInitiativesAPI) Delete(_ context.Context, _ string, _ *armpolicy.SetDefinitionsClientDeleteOptions) (armpolicy.SetDefinitionsClientDeleteResponse, error) {
	return armpolicy.SetDefinitionsClientDeleteResponse{}, nil
}

func (f *fakeInitiativesAPI) Get(_ context.Context, _ string, _ *armpolicy.SetDefinitionsClientGetOptions) (armpolicy.SetDefinitionsClientGetResponse, error) {
	return armpolicy.SetDefinitionsClientGetResponse{}, nil
}

func (f *fakeInitiativesAPI) CreateOrUpdateAtManagementGroup(_ context.Context, _ string, _ string, _ armpolicy.SetDefinition, _ *armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupOptions) (armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse, error) {
	return armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse{}, nil
}

func (f *fakeInitiativesAPI) DeleteAtManagementGroup(_ context.Context, _ string, _ string, _ *armpolicy.SetDefinitionsClientDeleteAtManagementGroupOptions) (armpolicy.SetDefinitionsClientDeleteAtManagementGroupResponse, error) {
	return armpolicy.SetDefinitionsClientDeleteAtManagementGroupResponse{}, nil
}

func (f *fakeInitiativesAPI) GetAtManagementGroup(_ context.Context, _ string, _ string, _ *armpolicy.SetDefinitionsClientGetAtManagementGroupOptions) (armpolicy.SetDefinitionsClientGetAtManagementGroupResponse, error) {
	return armpolicy.SetDefinitionsClientGetAtManagementGroupResponse{}, nil
}

// fakeAssignmentsAPI is a test double for assignments.API.
type fakeAssignmentsAPI struct{}

func (f *fakeAssignmentsAPI) Create(_ context.Context, _ string, _ string, _ armpolicy.Assignment, _ *armpolicy.AssignmentsClientCreateOptions) (armpolicy.AssignmentsClientCreateResponse, error) {
	return armpolicy.AssignmentsClientCreateResponse{}, nil
}

func (f *fakeAssignmentsAPI) Delete(_ context.Context, _ string, _ string, _ *armpolicy.AssignmentsClientDeleteOptions) (armpolicy.AssignmentsClientDeleteResponse, error) {
	return armpolicy.AssignmentsClientDeleteResponse{}, nil
}

func (f *fakeAssignmentsAPI) Get(_ context.Context, _ string, _ string, _ *armpolicy.AssignmentsClientGetOptions) (armpolicy.AssignmentsClientGetResponse, error) {
	return armpolicy.AssignmentsClientGetResponse{}, nil
}

func (f *fakeAssignmentsAPI) GetByID(_ context.Context, _ string, _ *armpolicy.AssignmentsClientGetByIDOptions) (armpolicy.AssignmentsClientGetByIDResponse, error) {
	return armpolicy.AssignmentsClientGetByIDResponse{}, nil
}

// fakeExemptionsAPI is a test double for exemptions.API.
type fakeExemptionsAPI struct{}

func (f *fakeExemptionsAPI) CreateOrUpdate(_ context.Context, _ string, _ string, _ armpolicy.Exemption, _ *armpolicy.ExemptionsClientCreateOrUpdateOptions) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
	return armpolicy.ExemptionsClientCreateOrUpdateResponse{}, nil
}

func (f *fakeExemptionsAPI) Delete(_ context.Context, _ string, _ string, _ *armpolicy.ExemptionsClientDeleteOptions) (armpolicy.ExemptionsClientDeleteResponse, error) {
	return armpolicy.ExemptionsClientDeleteResponse{}, nil
}

func (f *fakeExemptionsAPI) Get(_ context.Context, _ string, _ string, _ *armpolicy.ExemptionsClientGetOptions) (armpolicy.ExemptionsClientGetResponse, error) {
	return armpolicy.ExemptionsClientGetResponse{}, nil
}

// compile-time interface checks
var _ definitions.API = (*fakeDefinitionsAPI)(nil)
var _ initiatives.API = (*fakeInitiativesAPI)(nil)
var _ assignments.API = (*fakeAssignmentsAPI)(nil)
var _ exemptions.API = (*fakeExemptionsAPI)(nil)

// ── New() tests ──────────────────────────────────────────────────────────────

func TestNew_ReturnsNonNilClient(t *testing.T) {
	c, err := client.New("test-subscription-id")
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("New() returned nil ARMClient")
	}
}

func TestNew_SubscriptionIDIsPreserved(t *testing.T) {
	const sub = "aaaabbbb-cccc-dddd-eeee-ffffffffffff"

	c, err := client.New(sub)
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	if c.SubscriptionID != sub {
		t.Errorf("SubscriptionID: got %q, want %q", c.SubscriptionID, sub)
	}
}

func TestNew_AllFieldsInitialized(t *testing.T) {
	c, err := client.New("test-subscription-id")
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}

	if c.Definitions == nil {
		t.Error("Definitions field is nil")
	}
	if c.Initiatives == nil {
		t.Error("Initiatives field is nil")
	}
	if c.Assignments == nil {
		t.Error("Assignments field is nil")
	}
	if c.Exemptions == nil {
		t.Error("Exemptions field is nil")
	}
	if c.RoleAssignments == nil {
		t.Error("RoleAssignments field is nil")
	}
	if c.RoleDefinitions == nil {
		t.Error("RoleDefinitions field is nil")
	}
}

func TestNew_EmptySubscriptionID(t *testing.T) {
	c, err := client.New("")
	if err != nil {
		t.Fatalf("New() returned unexpected error: %v", err)
	}
	if c.SubscriptionID != "" {
		t.Errorf("SubscriptionID: got %q, want empty string", c.SubscriptionID)
	}
}

// ── ARMClient struct tests ────────────────────────────────────────────────────

func TestARMClient_DirectConstruction_SubscriptionID(t *testing.T) {
	const sub = "direct-construction-sub"

	c := &client.ARMClient{
		SubscriptionID: sub,
	}

	if c.SubscriptionID != sub {
		t.Errorf("SubscriptionID: got %q, want %q", c.SubscriptionID, sub)
	}
}

func TestARMClient_DirectConstruction_WithMockAPIs(t *testing.T) {
	c := &client.ARMClient{
		SubscriptionID: "mock-subscription",
		Definitions:    &fakeDefinitionsAPI{},
		Initiatives:    &fakeInitiativesAPI{},
		Assignments:    &fakeAssignmentsAPI{},
		Exemptions:     &fakeExemptionsAPI{},
	}

	if c.Definitions == nil {
		t.Error("Definitions field is nil")
	}
	if c.Initiatives == nil {
		t.Error("Initiatives field is nil")
	}
	if c.Assignments == nil {
		t.Error("Assignments field is nil")
	}
	if c.Exemptions == nil {
		t.Error("Exemptions field is nil")
	}
}

func TestARMClient_DirectConstruction_NilAPIsAllowed(t *testing.T) {
	// ARMClient with nil interface fields is valid at construction time;
	// callers are responsible for providing implementations before use.
	c := &client.ARMClient{
		SubscriptionID: "sub",
	}

	if c.Definitions != nil {
		t.Error("expected nil Definitions")
	}
	if c.Initiatives != nil {
		t.Error("expected nil Initiatives")
	}
	if c.Assignments != nil {
		t.Error("expected nil Assignments")
	}
	if c.Exemptions != nil {
		t.Error("expected nil Exemptions")
	}
}

func TestARMClient_DirectConstruction_OverrideField(t *testing.T) {
	c, err := client.New("original-sub")
	if err != nil {
		t.Fatalf("New() unexpected error: %v", err)
	}

	fake := &fakeDefinitionsAPI{}
	c.Definitions = fake

	if c.Definitions != fake {
		t.Error("expected Definitions to be the fake implementation after override")
	}
}
