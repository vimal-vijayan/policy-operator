package policyexemption

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// fakeExemptionsAPI is a manual fake for exemptions.API.
type fakeExemptionsAPI struct {
	createOrUpdateFn func(ctx context.Context, scope, name string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error)
	deleteFn         func(ctx context.Context, scope, name string) error
}

func (f *fakeExemptionsAPI) CreateOrUpdate(ctx context.Context, scope, name string, params armpolicy.Exemption, _ *armpolicy.ExemptionsClientCreateOrUpdateOptions) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, scope, name, params)
	}
	return armpolicy.ExemptionsClientCreateOrUpdateResponse{}, nil
}

func (f *fakeExemptionsAPI) Delete(ctx context.Context, scope, name string, _ *armpolicy.ExemptionsClientDeleteOptions) (armpolicy.ExemptionsClientDeleteResponse, error) {
	if f.deleteFn != nil {
		return armpolicy.ExemptionsClientDeleteResponse{}, f.deleteFn(ctx, scope, name)
	}
	return armpolicy.ExemptionsClientDeleteResponse{}, nil
}

func (f *fakeExemptionsAPI) Get(_ context.Context, _, _ string, _ *armpolicy.ExemptionsClientGetOptions) (armpolicy.ExemptionsClientGetResponse, error) {
	return armpolicy.ExemptionsClientGetResponse{}, nil
}

// helpers

func newTestExemptionService(api *fakeExemptionsAPI) *Service {
	return NewService(&client.ARMClient{Exemptions: api})
}

func newExemption(spec governancev1alpha1.AzurePolicyExemptionSpec) *governancev1alpha1.AzurePolicyExemption {
	return &governancev1alpha1.AzurePolicyExemption{
		ObjectMeta: metav1.ObjectMeta{Name: "test-exemption"},
		Spec:       spec,
	}
}

func ptrStr(s string) *string { return &s }

// ── CreateOrUpdate ──────────────────────────────────────────────────────────

func TestCreateOrUpdate_NoStatusID_GeneratesUUIDName(t *testing.T) {
	ctx := context.Background()

	var gotName string
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, name string, _ armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotName = name
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Authorization/policyExemptions/" + name)},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "My Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
	})

	id, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName == "" {
		t.Fatal("expected a generated name, got empty string")
	}
	// UUID format check: 36 chars with dashes
	if len(gotName) != 36 {
		t.Fatalf("expected UUID-format name (36 chars), got %q (len %d)", gotName, len(gotName))
	}
	if !strings.Contains(id, gotName) {
		t.Fatalf("expected returned ID to contain generated name %q, got %q", gotName, id)
	}
}

func TestCreateOrUpdate_WithStatusID_ReusesParsedName(t *testing.T) {
	ctx := context.Background()
	const existingID = "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Authorization/policyExemptions/existing-name"

	var gotName string
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, name string, _ armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotName = name
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr(existingID)},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "My Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
	})
	exemption.Status.ExemptionID = existingID

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "existing-name" {
		t.Fatalf("expected name %q, got %q", "existing-name", gotName)
	}
}

func TestCreateOrUpdate_SetsDisplayNameAndCategory(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "My Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Mitigated",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties == nil {
		t.Fatal("expected params.Properties to be set")
	}
	if *gotParams.Properties.DisplayName != "My Exemption" {
		t.Fatalf("expected DisplayName %q, got %q", "My Exemption", *gotParams.Properties.DisplayName)
	}
	if *gotParams.Properties.ExemptionCategory != armpolicy.ExemptionCategoryMitigated {
		t.Fatalf("expected ExemptionCategory Mitigated, got %v", *gotParams.Properties.ExemptionCategory)
	}
}

func TestCreateOrUpdate_SetsPolicyAssignmentID(t *testing.T) {
	ctx := context.Background()
	const assignmentID = "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1"

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: assignmentID,
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if *gotParams.Properties.PolicyAssignmentID != assignmentID {
		t.Fatalf("expected PolicyAssignmentID %q, got %q", assignmentID, *gotParams.Properties.PolicyAssignmentID)
	}
}

func TestCreateOrUpdate_SetsDescription(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
		Description:        "test description",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.Description == nil || *gotParams.Properties.Description != "test description" {
		t.Fatalf("expected description to be set, got %v", gotParams.Properties.Description)
	}
}

func TestCreateOrUpdate_EmptyDescription_NotSet(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.Description != nil {
		t.Fatalf("expected description to be nil when not set, got %v", *gotParams.Properties.Description)
	}
}

func TestCreateOrUpdate_ParsesValidExpiresOn(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
		ExpiresOn:          "2027-01-01T00:00:00Z",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.ExpiresOn == nil {
		t.Fatal("expected ExpiresOn to be set")
	}
	if gotParams.Properties.ExpiresOn.Year() != 2027 {
		t.Fatalf("expected year 2027, got %d", gotParams.Properties.ExpiresOn.Year())
	}
}

func TestCreateOrUpdate_InvalidExpiresOn_Ignored(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
		ExpiresOn:          "not-a-date",
	})

	// Should not return an error — invalid date is silently ignored
	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.ExpiresOn != nil {
		t.Fatal("expected ExpiresOn to be nil for invalid date input")
	}
}

func TestCreateOrUpdate_PassesScopeToAPI(t *testing.T) {
	ctx := context.Background()

	var gotScope string
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, scope, _ string, _ armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotScope = scope
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/my-rg",
		ExemptionCategory:  "Waiver",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotScope != "/subscriptions/sub1/resourceGroups/my-rg" {
		t.Fatalf("expected scope %q, got %q", "/subscriptions/sub1/resourceGroups/my-rg", gotScope)
	}
}

func TestCreateOrUpdate_SetsResourceSelectors_In(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
		ResourceSelectors: []governancev1alpha1.ResourceSelectorSpec{
			{
				Name: "selector1",
				Selectors: []governancev1alpha1.SelectorSpec{
					{Property: "resourceType", Operator: "In", Values: []string{"Microsoft.Compute/virtualMachines"}},
				},
			},
		},
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotParams.Properties.ResourceSelectors) != 1 {
		t.Fatalf("expected 1 resource selector, got %d", len(gotParams.Properties.ResourceSelectors))
	}
	rs := gotParams.Properties.ResourceSelectors[0]
	if *rs.Name != "selector1" {
		t.Fatalf("expected selector name %q, got %q", "selector1", *rs.Name)
	}
	if len(rs.Selectors) != 1 {
		t.Fatalf("expected 1 selector, got %d", len(rs.Selectors))
	}
	sel := rs.Selectors[0]
	if len(sel.In) != 1 || *sel.In[0] != "Microsoft.Compute/virtualMachines" {
		t.Fatalf("expected In selector with value, got %v", sel.In)
	}
	if len(sel.NotIn) != 0 {
		t.Fatalf("expected NotIn to be empty, got %v", sel.NotIn)
	}
}

func TestCreateOrUpdate_SetsResourceSelectors_NotIn(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
		ResourceSelectors: []governancev1alpha1.ResourceSelectorSpec{
			{
				Name: "sel2",
				Selectors: []governancev1alpha1.SelectorSpec{
					{Property: "resourceLocation", Operator: "notIn", Values: []string{"eastus", "westus"}},
				},
			},
		},
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sel := gotParams.Properties.ResourceSelectors[0].Selectors[0]
	if len(sel.NotIn) != 2 {
		t.Fatalf("expected 2 NotIn values, got %d", len(sel.NotIn))
	}
	if *sel.NotIn[0] != "eastus" || *sel.NotIn[1] != "westus" {
		t.Fatalf("unexpected NotIn values: %v, %v", *sel.NotIn[0], *sel.NotIn[1])
	}
}

func TestCreateOrUpdate_NoResourceSelectors_NotSet(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Exemption
	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, params armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{ID: ptrStr("id")},
			}, nil
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotParams.Properties.ResourceSelectors) != 0 {
		t.Fatalf("expected no resource selectors, got %d", len(gotParams.Properties.ResourceSelectors))
	}
}

func TestCreateOrUpdate_PropagatesAPIError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("azure api error")

	api := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, _, _ string, _ armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{}, expectedErr
		},
	}

	exemption := newExemption(governancev1alpha1.AzurePolicyExemptionSpec{
		DisplayName:        "Exemption",
		PolicyAssignmentID: "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/pa1",
		Scope:              "/subscriptions/sub1/resourceGroups/rg1",
		ExemptionCategory:  "Waiver",
	})

	if _, err := newTestExemptionService(api).CreateOrUpdate(ctx, exemption); !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

// ── Delete ──────────────────────────────────────────────────────────────────

func TestDelete_ExtractsNameFromIDAndPassesScope(t *testing.T) {
	ctx := context.Background()
	const exemptionID = "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Authorization/policyExemptions/my-exemption"
	const scope = "/subscriptions/sub1/resourceGroups/rg1"

	var gotScope, gotName string
	api := &fakeExemptionsAPI{
		deleteFn: func(_ context.Context, s, n string) error {
			gotScope = s
			gotName = n
			return nil
		},
	}

	if err := newTestExemptionService(api).Delete(ctx, scope, exemptionID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotScope != scope {
		t.Fatalf("expected scope %q, got %q", scope, gotScope)
	}
	if gotName != "my-exemption" {
		t.Fatalf("expected name %q, got %q", "my-exemption", gotName)
	}
}

func TestDelete_PropagatesAPIError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("delete error")

	api := &fakeExemptionsAPI{
		deleteFn: func(_ context.Context, _, _ string) error {
			return expectedErr
		},
	}

	err := newTestExemptionService(api).Delete(ctx, "/subscriptions/sub1/resourceGroups/rg1", "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.Authorization/policyExemptions/ex1")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}
