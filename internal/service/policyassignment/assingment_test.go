package policyassignment

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
	"github.com/vimal-vijayan/azure-policy-operator/internal/service/policyexemption"
	"k8s.io/apimachinery/pkg/runtime"
)

type fakeAssignmentsAPI struct {
	createFn  func(ctx context.Context, scope string, policyAssignmentName string, parameters armpolicy.Assignment) (armpolicy.AssignmentsClientCreateResponse, error)
	deleteFn  func(ctx context.Context, scope string, policyAssignmentName string) error
	getByIDFn func() (armpolicy.AssignmentsClientGetByIDResponse, error)
}

func (f *fakeAssignmentsAPI) Create(ctx context.Context, scope string, policyAssignmentName string, parameters armpolicy.Assignment, _ *armpolicy.AssignmentsClientCreateOptions) (armpolicy.AssignmentsClientCreateResponse, error) {
	if f.createFn != nil {
		return f.createFn(ctx, scope, policyAssignmentName, parameters)
	}
	return armpolicy.AssignmentsClientCreateResponse{}, nil
}

func (f *fakeAssignmentsAPI) Delete(ctx context.Context, scope string, policyAssignmentName string, _ *armpolicy.AssignmentsClientDeleteOptions) (armpolicy.AssignmentsClientDeleteResponse, error) {
	if f.deleteFn != nil {
		return armpolicy.AssignmentsClientDeleteResponse{}, f.deleteFn(ctx, scope, policyAssignmentName)
	}
	return armpolicy.AssignmentsClientDeleteResponse{}, nil
}

func (f *fakeAssignmentsAPI) Get(_ context.Context, _ string, _ string, _ *armpolicy.AssignmentsClientGetOptions) (armpolicy.AssignmentsClientGetResponse, error) {
	return armpolicy.AssignmentsClientGetResponse{}, nil
}

func (f *fakeAssignmentsAPI) GetByID(_ context.Context, _ string, _ *armpolicy.AssignmentsClientGetByIDOptions) (armpolicy.AssignmentsClientGetByIDResponse, error) {
	if f.getByIDFn != nil {
		return f.getByIDFn()
	}
	return armpolicy.AssignmentsClientGetByIDResponse{}, nil
}

type fakeExemptionsAPI struct {
	createOrUpdateFn func(ctx context.Context, scope string, policyExemptionName string, parameters armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error)
	deleteFn         func(ctx context.Context, scope string, policyExemptionName string) error
}

func (f *fakeExemptionsAPI) CreateOrUpdate(ctx context.Context, scope string, policyExemptionName string, parameters armpolicy.Exemption, _ *armpolicy.ExemptionsClientCreateOrUpdateOptions) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, scope, policyExemptionName, parameters)
	}
	return armpolicy.ExemptionsClientCreateOrUpdateResponse{}, nil
}

func (f *fakeExemptionsAPI) Delete(ctx context.Context, scope string, policyExemptionName string, _ *armpolicy.ExemptionsClientDeleteOptions) (armpolicy.ExemptionsClientDeleteResponse, error) {
	if f.deleteFn != nil {
		return armpolicy.ExemptionsClientDeleteResponse{}, f.deleteFn(ctx, scope, policyExemptionName)
	}
	return armpolicy.ExemptionsClientDeleteResponse{}, nil
}

func (f *fakeExemptionsAPI) Get(_ context.Context, _ string, _ string, _ *armpolicy.ExemptionsClientGetOptions) (armpolicy.ExemptionsClientGetResponse, error) {
	return armpolicy.ExemptionsClientGetResponse{}, nil
}

func TestCreateOrUpdate_UsesExistingAssignmentName_ParsesParametersAndPreservesLocation(t *testing.T) {
	ctx := context.Background()

	var gotCreateScope, gotCreateName string
	var gotParams armpolicy.Assignment

	fakeAssignments := &fakeAssignmentsAPI{
		createFn: func(_ context.Context, scope string, policyAssignmentName string, parameters armpolicy.Assignment) (armpolicy.AssignmentsClientCreateResponse, error) {
			gotCreateScope = scope
			gotCreateName = policyAssignmentName
			gotParams = parameters

			return armpolicy.AssignmentsClientCreateResponse{
				Assignment: armpolicy.Assignment{
					ID:       to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/existing-name"),
					Location: to.Ptr("westeurope"),
				},
			}, nil
		},
	}

	fakeExemptions := &fakeExemptionsAPI{}

	arm := &client.ARMClient{
		Assignments: fakeAssignments,
		Exemptions:  fakeExemptions,
	}

	exemptionSvc := policyexemption.NewService(arm)
	svc := NewService(arm, exemptionSvc)

	assignment := &governancev1alpha1.AzurePolicyAssignment{
		Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
			DisplayName: "assignment-display",
			Scope:       "/subscriptions/sub1",
			Parameters: &runtime.RawExtension{
				Raw: []byte(`{"effect":"Audit"}`),
			},
		},
		Status: governancev1alpha1.AzurePolicyAssignmentStatus{
			AssignmentID:     "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/existing-name",
			AssignedLocation: "eastus",
		},
	}

	assignmentID, assignedLocation, principalID, exemptions, err := svc.CreateOrUpdate(ctx, assignment, "/providers/Microsoft.Authorization/policyDefinitions/def1")
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}

	if gotCreateScope != assignment.Spec.Scope {
		t.Fatalf("expected scope %q, got %q", assignment.Spec.Scope, gotCreateScope)
	}

	if gotCreateName != "existing-name" {
		t.Fatalf("expected existing assignment name to be reused, got %q", gotCreateName)
	}

	if gotParams.Location == nil || *gotParams.Location != "eastus" {
		t.Fatalf("expected location to be preserved as eastus, got %#v", gotParams.Location)
	}

	if gotParams.Properties == nil || gotParams.Properties.Parameters == nil {
		t.Fatalf("expected assignment parameters to be populated")
	}

	if gotParams.Properties.Parameters["effect"] == nil || gotParams.Properties.Parameters["effect"].Value != "Audit" {
		t.Fatalf("expected wrapped flat parameter value, got %#v", gotParams.Properties.Parameters["effect"])
	}

	if assignmentID != "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/existing-name" {
		t.Fatalf("unexpected assignmentID %q", assignmentID)
	}

	if assignedLocation != "westeurope" {
		t.Fatalf("unexpected assignedLocation %q", assignedLocation)
	}

	if principalID != "" {
		t.Fatalf("expected empty principalID, got %q", principalID)
	}

	if len(exemptions) != 0 {
		t.Fatalf("expected no exemptions status, got %d", len(exemptions))
	}
}

func TestCreateOrUpdate_ReconcilesInlineExemptions(t *testing.T) {
	ctx := context.Background()

	fakeAssignments := &fakeAssignmentsAPI{
		createFn: func(_ context.Context, _, _ string, _ armpolicy.Assignment) (armpolicy.AssignmentsClientCreateResponse, error) {
			return armpolicy.AssignmentsClientCreateResponse{
				Assignment: armpolicy.Assignment{
					ID: to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/assignment-a"),
				},
			}, nil
		},
	}

	createCalls := make(map[string]string)
	deleteCalls := make(map[string]string)

	fakeExemptions := &fakeExemptionsAPI{
		createOrUpdateFn: func(_ context.Context, scope string, policyExemptionName string, parameters armpolicy.Exemption) (armpolicy.ExemptionsClientCreateOrUpdateResponse, error) {
			display := ""
			if parameters.Properties != nil && parameters.Properties.DisplayName != nil {
				display = *parameters.Properties.DisplayName
			}
			createCalls[display] = policyExemptionName
			return armpolicy.ExemptionsClientCreateOrUpdateResponse{
				Exemption: armpolicy.Exemption{
					ID: to.Ptr(fmt.Sprintf("%s/providers/Microsoft.Authorization/policyExemptions/%s", scope, policyExemptionName)),
				},
			}, nil
		},
		deleteFn: func(_ context.Context, scope string, policyExemptionName string) error {
			deleteCalls[scope] = policyExemptionName
			return nil
		},
	}

	arm := &client.ARMClient{
		Assignments: fakeAssignments,
		Exemptions:  fakeExemptions,
	}

	exemptionSvc := policyexemption.NewService(arm)
	svc := NewService(arm, exemptionSvc)

	assignment := &governancev1alpha1.AzurePolicyAssignment{
		Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
			DisplayName: "assignment-display",
			Scope:       "/subscriptions/sub1",
			Exemptions: []governancev1alpha1.AssignmentExemptionSpec{
				{
					DisplayName:       "keep-existing",
					Scope:             "/subscriptions/sub1/resourceGroups/rg-a",
					ExemptionCategory: "Waiver",
				},
				{
					DisplayName:       "new-exemption",
					Scope:             "/subscriptions/sub1/resourceGroups/rg-b",
					ExemptionCategory: "Waiver",
				},
			},
		},
		Status: governancev1alpha1.AzurePolicyAssignmentStatus{
			Exemptions: []governancev1alpha1.AssignmentExemptionStatus{
				{
					DisplayName: "keep-existing",
					ExemptionID: "/subscriptions/sub1/resourceGroups/rg-a/providers/Microsoft.Authorization/policyExemptions/existing-exemption-name",
					Scope:       "/subscriptions/sub1/resourceGroups/rg-a",
				},
				{
					DisplayName: "removed-exemption",
					ExemptionID: "/subscriptions/sub1/resourceGroups/rg-c/providers/Microsoft.Authorization/policyExemptions/removed-exemption-name",
					Scope:       "/subscriptions/sub1/resourceGroups/rg-c",
				},
			},
		},
	}

	_, _, _, statuses, err := svc.CreateOrUpdate(ctx, assignment, "/providers/Microsoft.Authorization/policyDefinitions/def1")
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}

	if createCalls["keep-existing"] != "existing-exemption-name" {
		t.Fatalf("expected existing exemption name to be reused, got %q", createCalls["keep-existing"])
	}

	if _, ok := createCalls["new-exemption"]; !ok {
		t.Fatalf("expected new exemption to be created")
	}

	if deleteCalls["/subscriptions/sub1/resourceGroups/rg-c"] != "removed-exemption-name" {
		t.Fatalf("expected removed exemption to be deleted, got %q", deleteCalls["/subscriptions/sub1/resourceGroups/rg-c"])
	}

	if len(statuses) != 2 {
		t.Fatalf("expected 2 exemption statuses, got %d", len(statuses))
	}
}

func TestCreateOrUpdate_AssignmentUpdateWithUserAssignedIdentity(t *testing.T) {
	ctx := context.Background()

	var gotCreateName string
	var gotParams armpolicy.Assignment

	fakeAssignments := &fakeAssignmentsAPI{
		createFn: func(_ context.Context, _ string, policyAssignmentName string, parameters armpolicy.Assignment) (armpolicy.AssignmentsClientCreateResponse, error) {
			gotCreateName = policyAssignmentName
			gotParams = parameters

			return armpolicy.AssignmentsClientCreateResponse{
				Assignment: armpolicy.Assignment{
					ID:       to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/existing-assignment"),
					Location: to.Ptr("eastus2"),
					Identity: &armpolicy.Identity{
						PrincipalID: to.Ptr("mi-principal-id"),
					},
				},
			}, nil
		},
	}

	fakeExemptions := &fakeExemptionsAPI{}

	arm := &client.ARMClient{
		Assignments: fakeAssignments,
		Exemptions:  fakeExemptions,
	}

	exemptionSvc := policyexemption.NewService(arm)
	svc := NewService(arm, exemptionSvc)

	assignment := &governancev1alpha1.AzurePolicyAssignment{
		Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
			DisplayName: "assignment-display",
			Scope:       "/subscriptions/sub1",
			Identity: &governancev1alpha1.AssignmentIdentity{
				Type:                   "UserAssigned",
				Location:               "westeurope",
				UserAssignedIdentityID: "/subscriptions/sub1/resourceGroups/rg1/providers/Microsoft.ManagedIdentity/userAssignedIdentities/id1",
			},
		},
		Status: governancev1alpha1.AzurePolicyAssignmentStatus{
			AssignmentID:     "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/existing-assignment",
			AssignedLocation: "eastus",
		},
	}

	assignmentID, assignedLocation, principalID, _, err := svc.CreateOrUpdate(ctx, assignment, "/providers/Microsoft.Authorization/policyDefinitions/def1")
	if err != nil {
		t.Fatalf("CreateOrUpdate returned error: %v", err)
	}

	if gotCreateName != "existing-assignment" {
		t.Fatalf("expected existing assignment name to be reused, got %q", gotCreateName)
	}

	if gotParams.Identity == nil || gotParams.Identity.Type == nil || *gotParams.Identity.Type != armpolicy.ResourceIdentityTypeUserAssigned {
		t.Fatalf("expected user-assigned identity to be configured, got %#v", gotParams.Identity)
	}

	if gotParams.Location == nil || *gotParams.Location != "westeurope" {
		t.Fatalf("expected assignment location from identity settings, got %#v", gotParams.Location)
	}

	if gotParams.Identity.UserAssignedIdentities == nil {
		t.Fatalf("expected userAssignedIdentities to be set")
	}

	if _, ok := gotParams.Identity.UserAssignedIdentities[assignment.Spec.Identity.UserAssignedIdentityID]; !ok {
		t.Fatalf("expected userAssignedIdentityId %q to be present", assignment.Spec.Identity.UserAssignedIdentityID)
	}

	if assignmentID != "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/existing-assignment" {
		t.Fatalf("unexpected assignmentID %q", assignmentID)
	}

	if assignedLocation != "eastus2" {
		t.Fatalf("unexpected assignedLocation %q", assignedLocation)
	}

	if principalID != "mi-principal-id" {
		t.Fatalf("unexpected principalID %q", principalID)
	}
}

func TestCreateOrUpdate_ReturnsErrorForInvalidIdentityPermission(t *testing.T) {
	ctx := context.Background()

	fakeAssignments := &fakeAssignmentsAPI{
		createFn: func(_ context.Context, _, _ string, _ armpolicy.Assignment) (armpolicy.AssignmentsClientCreateResponse, error) {
			return armpolicy.AssignmentsClientCreateResponse{
				Assignment: armpolicy.Assignment{
					ID: to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/assignment-a"),
					Identity: &armpolicy.Identity{
						PrincipalID: to.Ptr("principal-1"),
					},
				},
			}, nil
		},
	}

	fakeExemptions := &fakeExemptionsAPI{}

	arm := &client.ARMClient{
		Assignments: fakeAssignments,
		Exemptions:  fakeExemptions,
	}

	exemptionSvc := policyexemption.NewService(arm)
	svc := NewService(arm, exemptionSvc)

	assignment := &governancev1alpha1.AzurePolicyAssignment{
		Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
			DisplayName: "assignment-display",
			Scope:       "/subscriptions/sub1",
			Identity: &governancev1alpha1.AssignmentIdentity{
				Type: "SystemAssigned",
				Permissions: []governancev1alpha1.IdentityPermission{
					{
						Scope: "/subscriptions/sub1",
					},
				},
			},
		},
	}

	_, _, _, _, err := svc.CreateOrUpdate(ctx, assignment, "/providers/Microsoft.Authorization/policyDefinitions/def1")
	if err == nil {
		t.Fatalf("expected error for invalid identity permission, got nil")
	}

	expected := "either role or roleDefinitionId must be specified"
	if err != nil && !strings.Contains(err.Error(), expected) {
		t.Fatalf("expected error to contain %q, got %q", expected, err.Error())
	}
}

func TestDelete_DeletesExemptionsThenAssignment(t *testing.T) {
	ctx := context.Background()

	deletedExemptions := make(map[string]string)
	var deletedAssignmentScope, deletedAssignmentName string

	fakeAssignments := &fakeAssignmentsAPI{
		deleteFn: func(_ context.Context, scope string, policyAssignmentName string) error {
			deletedAssignmentScope = scope
			deletedAssignmentName = policyAssignmentName
			return nil
		},
	}

	fakeExemptions := &fakeExemptionsAPI{
		deleteFn: func(_ context.Context, scope string, policyExemptionName string) error {
			deletedExemptions[scope] = policyExemptionName
			return nil
		},
	}

	arm := &client.ARMClient{
		Assignments: fakeAssignments,
		Exemptions:  fakeExemptions,
	}

	exemptionSvc := policyexemption.NewService(arm)
	svc := NewService(arm, exemptionSvc)

	assignmentID := "/subscriptions/sub1/providers/Microsoft.Authorization/policyAssignments/assignment-a"
	exemptions := []governancev1alpha1.AssignmentExemptionStatus{
		{
			DisplayName: "e1",
			ExemptionID: "/subscriptions/sub1/resourceGroups/rg-a/providers/Microsoft.Authorization/policyExemptions/ex1",
			Scope:       "/subscriptions/sub1/resourceGroups/rg-a",
		},
		{
			DisplayName: "e2",
			ExemptionID: "/subscriptions/sub1/resourceGroups/rg-b/providers/Microsoft.Authorization/policyExemptions/ex2",
			Scope:       "/subscriptions/sub1/resourceGroups/rg-b",
		},
	}

	err := svc.Delete(ctx, "/subscriptions/sub1", assignmentID, exemptions, nil)
	if err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}

	if deletedExemptions["/subscriptions/sub1/resourceGroups/rg-a"] != "ex1" {
		t.Fatalf("expected ex1 to be deleted at rg-a")
	}

	if deletedExemptions["/subscriptions/sub1/resourceGroups/rg-b"] != "ex2" {
		t.Fatalf("expected ex2 to be deleted at rg-b")
	}

	if deletedAssignmentScope != "/subscriptions/sub1" {
		t.Fatalf("unexpected assignment delete scope %q", deletedAssignmentScope)
	}

	if deletedAssignmentName != "assignment-a" {
		t.Fatalf("unexpected assignment name %q", deletedAssignmentName)
	}
}
