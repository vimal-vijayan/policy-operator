package policybundle

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
)

const auditEffect = "Audit"

// fakeInitiativesAPI implements initiatives.API using in-memory functions.
type fakeInitiativesAPI struct {
	createOrUpdateFn            func(ctx context.Context, name string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error)
	createOrUpdateAtMgmtGroupFn func(ctx context.Context, mgmtGroup, name string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse, error)
	deleteFn                    func(ctx context.Context, name string) error
	deleteAtMgmtGroupFn         func(ctx context.Context, mgmtGroup, name string) error
}

func (f *fakeInitiativesAPI) CreateOrUpdate(ctx context.Context, name string, params armpolicy.SetDefinition, _ *armpolicy.SetDefinitionsClientCreateOrUpdateOptions) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, name, params)
	}
	return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{}, nil
}

func (f *fakeInitiativesAPI) CreateOrUpdateAtManagementGroup(ctx context.Context, mgmtGroup, name string, params armpolicy.SetDefinition, _ *armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupOptions) (armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse, error) {
	if f.createOrUpdateAtMgmtGroupFn != nil {
		return f.createOrUpdateAtMgmtGroupFn(ctx, mgmtGroup, name, params)
	}
	return armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse{}, nil
}

func (f *fakeInitiativesAPI) Delete(ctx context.Context, name string, _ *armpolicy.SetDefinitionsClientDeleteOptions) (armpolicy.SetDefinitionsClientDeleteResponse, error) {
	if f.deleteFn != nil {
		return armpolicy.SetDefinitionsClientDeleteResponse{}, f.deleteFn(ctx, name)
	}
	return armpolicy.SetDefinitionsClientDeleteResponse{}, nil
}

func (f *fakeInitiativesAPI) DeleteAtManagementGroup(ctx context.Context, mgmtGroup, name string, _ *armpolicy.SetDefinitionsClientDeleteAtManagementGroupOptions) (armpolicy.SetDefinitionsClientDeleteAtManagementGroupResponse, error) {
	if f.deleteAtMgmtGroupFn != nil {
		return armpolicy.SetDefinitionsClientDeleteAtManagementGroupResponse{}, f.deleteAtMgmtGroupFn(ctx, mgmtGroup, name)
	}
	return armpolicy.SetDefinitionsClientDeleteAtManagementGroupResponse{}, nil
}

func (f *fakeInitiativesAPI) Get(_ context.Context, _ string, _ *armpolicy.SetDefinitionsClientGetOptions) (armpolicy.SetDefinitionsClientGetResponse, error) {
	return armpolicy.SetDefinitionsClientGetResponse{}, nil
}

func (f *fakeInitiativesAPI) GetAtManagementGroup(_ context.Context, _, _ string, _ *armpolicy.SetDefinitionsClientGetAtManagementGroupOptions) (armpolicy.SetDefinitionsClientGetAtManagementGroupResponse, error) {
	return armpolicy.SetDefinitionsClientGetAtManagementGroupResponse{}, nil
}

func newTestService(api *fakeInitiativesAPI) *Service {
	return NewService(&client.ARMClient{Initiatives: api})
}

func newInitiative(name string, spec governancev1alpha1.AzurePolicyInitiativeSpec) *governancev1alpha1.AzurePolicyInitiative {
	return &governancev1alpha1.AzurePolicyInitiative{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       spec,
	}
}

// --- CreateOrUpdate ---

func TestCreateOrUpdate_SubscriptionScope_ReturnsInitiativeID(t *testing.T) {
	const fakeID = "/subscriptions/sub1/providers/Microsoft.Authorization/policySetDefinitions/my-initiative"
	ctx := context.Background()

	var gotName string
	var gotParams armpolicy.SetDefinition

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, name string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			gotName = name
			gotParams = params
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{
				SetDefinition: armpolicy.SetDefinition{ID: to.Ptr(fakeID)},
			}, nil
		},
	}

	initiative := newInitiative("my-initiative", governancev1alpha1.AzurePolicyInitiativeSpec{
		DisplayName: "My Initiative",
		Description: "test description",
	})

	id, err := newTestService(api).CreateOrUpdate(ctx, initiative, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != fakeID {
		t.Fatalf("expected ID %q, got %q", fakeID, id)
	}
	//nolint:goconst // Keep the expected name literal local to this assertion for test readability.
	if gotName != "my-initiative" {
		t.Fatalf("expected name %q, got %q", "my-initiative", gotName)
	}
	if gotParams.Properties == nil || *gotParams.Properties.DisplayName != "My Initiative" {
		t.Fatalf("unexpected display name: %#v", gotParams.Properties)
	}
	if gotParams.Properties.Description == nil || *gotParams.Properties.Description != "test description" {
		t.Fatalf("expected description to be set")
	}
}

func TestCreateOrUpdate_ManagementGroupScope_CallsManagementGroupAPI(t *testing.T) {
	const fakeID = "/providers/Microsoft.Management/managementGroups/mg1/providers/Microsoft.Authorization/policySetDefinitions/my-initiative"
	ctx := context.Background()

	var gotMgmtGroup string
	var gotName string
	subCalled := false

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, _ string, _ armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			subCalled = true
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{}, nil
		},
		createOrUpdateAtMgmtGroupFn: func(_ context.Context, mgmtGroup, name string, _ armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse, error) {
			gotMgmtGroup = mgmtGroup
			gotName = name
			return armpolicy.SetDefinitionsClientCreateOrUpdateAtManagementGroupResponse{
				SetDefinition: armpolicy.SetDefinition{ID: to.Ptr(fakeID)},
			}, nil
		},
	}

	initiative := newInitiative("my-initiative", governancev1alpha1.AzurePolicyInitiativeSpec{
		DisplayName:       "My Initiative",
		ManagementGroupID: "mg1",
	})

	id, err := newTestService(api).CreateOrUpdate(ctx, initiative, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != fakeID {
		t.Fatalf("expected ID %q, got %q", fakeID, id)
	}
	if gotMgmtGroup != "mg1" {
		t.Fatalf("expected management group %q, got %q", "mg1", gotMgmtGroup)
	}
	if gotName != "my-initiative" {
		t.Fatalf("expected initiative name %q, got %q", "my-initiative", gotName)
	}
	if subCalled {
		t.Fatalf("expected subscription-scope API NOT to be called")
	}
}

func TestCreateOrUpdate_PolicyDefinitionRefs_MappedInOrder(t *testing.T) {
	ctx := context.Background()

	resolvedIDs := []string{
		"/subscriptions/sub1/providers/Microsoft.Authorization/policyDefinitions/def-a",
		"/subscriptions/sub1/providers/Microsoft.Authorization/policyDefinitions/def-b",
	}

	var gotDefs []*armpolicy.DefinitionReference

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			gotDefs = params.Properties.PolicyDefinitions
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{
				SetDefinition: armpolicy.SetDefinition{ID: to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policySetDefinitions/i")},
			}, nil
		},
	}

	initiative := newInitiative("i", governancev1alpha1.AzurePolicyInitiativeSpec{
		DisplayName: "My Initiative",
		PolicyDefinitions: []governancev1alpha1.PolicyDefinitionReference{
			{PolicyDefinitionID: "ref-a"},
			{PolicyDefinitionID: "ref-b"},
		},
	})

	_, err := newTestService(api).CreateOrUpdate(ctx, initiative, resolvedIDs)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotDefs) != 2 {
		t.Fatalf("expected 2 policy definition refs, got %d", len(gotDefs))
	}
	if *gotDefs[0].PolicyDefinitionID != resolvedIDs[0] {
		t.Fatalf("expected first ref %q, got %q", resolvedIDs[0], *gotDefs[0].PolicyDefinitionID)
	}
	if *gotDefs[1].PolicyDefinitionID != resolvedIDs[1] {
		t.Fatalf("expected second ref %q, got %q", resolvedIDs[1], *gotDefs[1].PolicyDefinitionID)
	}
}

func TestCreateOrUpdate_PolicyDefinitionRef_WithParameters(t *testing.T) {
	ctx := context.Background()

	var gotDefs []*armpolicy.DefinitionReference

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			gotDefs = params.Properties.PolicyDefinitions
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{
				SetDefinition: armpolicy.SetDefinition{ID: to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policySetDefinitions/i")},
			}, nil
		},
	}

	initiative := newInitiative("i", governancev1alpha1.AzurePolicyInitiativeSpec{
		DisplayName: "My Initiative",
		PolicyDefinitions: []governancev1alpha1.PolicyDefinitionReference{
			{
				PolicyDefinitionID: "ref-a",
				Parameters:         &runtime.RawExtension{Raw: []byte(`{"effect":{"value":"Audit"}}`)},
			},
		},
	})

	_, err := newTestService(api).CreateOrUpdate(ctx, initiative, []string{"/subscriptions/sub1/providers/Microsoft.Authorization/policyDefinitions/def-a"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotDefs[0].Parameters == nil {
		t.Fatalf("expected parameters to be set on policy definition ref")
	}
	param, ok := gotDefs[0].Parameters["effect"]
	if !ok {
		t.Fatalf("expected 'effect' parameter to be present")
	}
	if param.Value != auditEffect {
		t.Fatalf("expected param value %q, got %#v", auditEffect, param.Value)
	}
}

func TestCreateOrUpdate_VersionInjectedIntoMetadata(t *testing.T) {
	ctx := context.Background()

	var gotMeta interface{}

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			gotMeta = params.Properties.Metadata
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{
				SetDefinition: armpolicy.SetDefinition{ID: to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policySetDefinitions/i")},
			}, nil
		},
	}

	initiative := newInitiative("i", governancev1alpha1.AzurePolicyInitiativeSpec{
		DisplayName: "My Initiative",
		Version:     "2.1.0",
	})

	_, err := newTestService(api).CreateOrUpdate(ctx, initiative, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	meta, ok := gotMeta.(map[string]interface{})
	if !ok {
		t.Fatalf("expected metadata to be a map, got %T", gotMeta)
	}
	if meta["version"] != "2.1.0" {
		t.Fatalf("expected version %q in metadata, got %#v", "2.1.0", meta["version"])
	}
}

func TestCreateOrUpdate_ReturnsError_WhenAPIFails(t *testing.T) {
	ctx := context.Background()

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, _ string, _ armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{}, errors.New("azure api error")
		},
	}

	_, err := newTestService(api).CreateOrUpdate(ctx, newInitiative("i", governancev1alpha1.AzurePolicyInitiativeSpec{DisplayName: "My Initiative"}), []string{})
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestCreateOrUpdate_NoDescription_DoesNotSetDescription(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.SetDefinition

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{
				SetDefinition: armpolicy.SetDefinition{ID: to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policySetDefinitions/i")},
			}, nil
		},
	}

	_, err := newTestService(api).CreateOrUpdate(ctx, newInitiative("i", governancev1alpha1.AzurePolicyInitiativeSpec{
		DisplayName: "My Initiative",
	}), []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.Description != nil {
		t.Fatalf("expected description to be nil, got %q", *gotParams.Properties.Description)
	}
}

func TestCreateOrUpdate_WithParameters_SetsParametersOnPayload(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.SetDefinition

	api := &fakeInitiativesAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.SetDefinition) (armpolicy.SetDefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.SetDefinitionsClientCreateOrUpdateResponse{
				SetDefinition: armpolicy.SetDefinition{ID: to.Ptr("/subscriptions/sub1/providers/Microsoft.Authorization/policySetDefinitions/i")},
			}, nil
		},
	}

	initiative := newInitiative("i", governancev1alpha1.AzurePolicyInitiativeSpec{
		DisplayName: "My Initiative",
		Parameters: []governancev1alpha1.InitiativeParameter{
			{Name: "effect", Type: "String"},
		},
	})

	_, err := newTestService(api).CreateOrUpdate(ctx, initiative, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.Parameters == nil {
		t.Fatalf("expected parameters map to be set")
	}
	if _, ok := gotParams.Properties.Parameters["effect"]; !ok {
		t.Fatalf("expected 'effect' key in parameters map")
	}
}

// --- Delete ---

func TestDelete_SubscriptionScope_UsesNameFromInitiativeID(t *testing.T) {
	ctx := context.Background()

	var deletedName string

	api := &fakeInitiativesAPI{
		deleteFn: func(_ context.Context, name string) error {
			deletedName = name
			return nil
		},
	}

	initiative := newInitiative("my-initiative", governancev1alpha1.AzurePolicyInitiativeSpec{})
	initiative.Status.InitiativeID = "/subscriptions/sub1/providers/Microsoft.Authorization/policySetDefinitions/stable-name"

	if err := newTestService(api).Delete(ctx, initiative); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedName != "stable-name" {
		t.Fatalf("expected %q derived from InitiativeID, got %q", "stable-name", deletedName)
	}
}

func TestDelete_SubscriptionScope_FallsBackToObjectName(t *testing.T) {
	ctx := context.Background()

	var deletedName string

	api := &fakeInitiativesAPI{
		deleteFn: func(_ context.Context, name string) error {
			deletedName = name
			return nil
		},
	}

	if err := newTestService(api).Delete(ctx, newInitiative("my-initiative", governancev1alpha1.AzurePolicyInitiativeSpec{})); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedName != "my-initiative" {
		t.Fatalf("expected fallback to object name %q, got %q", "my-initiative", deletedName)
	}
}

func TestDelete_ManagementGroupScope_CallsManagementGroupAPI(t *testing.T) {
	ctx := context.Background()

	var deletedName, deletedMgmtGroup string
	subCalled := false

	api := &fakeInitiativesAPI{
		deleteFn: func(_ context.Context, _ string) error {
			subCalled = true
			return nil
		},
		deleteAtMgmtGroupFn: func(_ context.Context, mgmtGroup, name string) error {
			deletedName = name
			deletedMgmtGroup = mgmtGroup
			return nil
		},
	}

	initiative := newInitiative("my-initiative", governancev1alpha1.AzurePolicyInitiativeSpec{ManagementGroupID: "mg1"})
	initiative.Status.InitiativeID = "/providers/Microsoft.Management/managementGroups/mg1/providers/Microsoft.Authorization/policySetDefinitions/stable-name"

	if err := newTestService(api).Delete(ctx, initiative); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedMgmtGroup != "mg1" {
		t.Fatalf("expected management group %q, got %q", "mg1", deletedMgmtGroup)
	}
	if deletedName != "stable-name" {
		t.Fatalf("expected name %q derived from InitiativeID, got %q", "stable-name", deletedName)
	}
	if subCalled {
		t.Fatalf("expected subscription-scope Delete NOT to be called")
	}
}

func TestDelete_ReturnsError_WhenAPIFails(t *testing.T) {
	ctx := context.Background()

	api := &fakeInitiativesAPI{
		deleteFn: func(_ context.Context, _ string) error {
			return errors.New("azure api error")
		},
	}

	if err := newTestService(api).Delete(ctx, newInitiative("i", governancev1alpha1.AzurePolicyInitiativeSpec{})); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

// --- buildMetadata unit tests ---

func TestBuildMetadata_NilMetaNoVersion_ReturnsNil(t *testing.T) {
	if result := buildMetadata(nil, ""); result != nil {
		t.Fatalf("expected nil, got %#v", result)
	}
}

func TestBuildMetadata_VersionOnly_InjectsVersion(t *testing.T) {
	meta, ok := buildMetadata(nil, "1.2.3").(map[string]interface{})
	if !ok {
		t.Fatalf("expected map")
	}
	if meta["version"] != "1.2.3" {
		t.Fatalf("expected version %q, got %#v", "1.2.3", meta["version"])
	}
}

func TestBuildMetadata_ExistingMetaWithVersion_MergesVersion(t *testing.T) {
	raw := &runtime.RawExtension{Raw: []byte(`{"category":"Security"}`)}
	meta, ok := buildMetadata(raw, "3.0.0").(map[string]interface{})
	if !ok {
		t.Fatalf("expected map")
	}
	if meta["category"] != "Security" {
		t.Fatalf("expected category to be preserved")
	}
	if meta["version"] != "3.0.0" {
		t.Fatalf("expected version to be injected")
	}
}

func TestBuildMetadata_ExistingMetaNoVersion_ReturnsExistingKeys(t *testing.T) {
	raw := &runtime.RawExtension{Raw: []byte(`{"category":"Compute","author":"team"}`)}
	meta, ok := buildMetadata(raw, "").(map[string]interface{})
	if !ok {
		t.Fatalf("expected map")
	}
	if meta["category"] != "Compute" {
		t.Fatalf("expected category %q, got %#v", "Compute", meta["category"])
	}
	if _, hasVersion := meta["version"]; hasVersion {
		t.Fatalf("expected no version key, got one")
	}
}

func TestBuildMetadata_InvalidJSON_ReturnsVersionOnly(t *testing.T) {
	raw := &runtime.RawExtension{Raw: []byte(`not-json`)}
	meta, ok := buildMetadata(raw, "1.0.0").(map[string]interface{})
	if !ok {
		t.Fatalf("expected map")
	}
	if meta["version"] != "1.0.0" {
		t.Fatalf("expected version %q after JSON parse failure, got %#v", "1.0.0", meta["version"])
	}
}

// --- buildPolicyDefinitionRefs unit tests ---

func TestBuildPolicyDefinitionRefs_EmptyParameters_NilParametersField(t *testing.T) {
	refs := []governancev1alpha1.PolicyDefinitionReference{
		{PolicyDefinitionID: "ref-a"},
	}
	result := buildPolicyDefinitionRefs(refs, []string{"azure-id-a"})
	if len(result) != 1 {
		t.Fatalf("expected 1 result")
	}
	if *result[0].PolicyDefinitionID != "azure-id-a" {
		t.Fatalf("expected resolved ID %q, got %q", "azure-id-a", *result[0].PolicyDefinitionID)
	}
	if result[0].Parameters != nil {
		t.Fatalf("expected nil parameters when none specified")
	}
}

func TestBuildPolicyDefinitionRefs_FlatParameterValue_UsedDirectly(t *testing.T) {
	// When the parameter JSON does NOT have a nested {"value": ...} structure,
	// the raw value is used directly.
	refs := []governancev1alpha1.PolicyDefinitionReference{
		{
			PolicyDefinitionID: "ref-a",
			Parameters:         &runtime.RawExtension{Raw: []byte(`{"mode":"Indexed"}`)},
		},
	}
	result := buildPolicyDefinitionRefs(refs, []string{"azure-id-a"})
	param, ok := result[0].Parameters["mode"]
	if !ok {
		t.Fatalf("expected 'mode' parameter")
	}
	if param.Value != "Indexed" {
		t.Fatalf("expected value %q, got %#v", "Indexed", param.Value)
	}
}

// --- buildParameters unit tests ---

func TestBuildParameters_Type_IsSet(t *testing.T) {
	params := []governancev1alpha1.InitiativeParameter{
		{Name: "effect", Type: "String"},
	}
	result := buildParameters(params)
	p, ok := result["effect"]
	if !ok {
		t.Fatalf("expected 'effect' key")
	}
	if p.Type == nil || *p.Type != armpolicy.ParameterTypeString {
		t.Fatalf("expected type String, got %v", p.Type)
	}
}

func TestBuildParameters_DefaultValue_IsUnmarshalled(t *testing.T) {
	params := []governancev1alpha1.InitiativeParameter{
		{
			Name:         "effect",
			Type:         "String",
			DefaultValue: &runtime.RawExtension{Raw: []byte(`"Audit"`)},
		},
	}
	result := buildParameters(params)
	p := result["effect"]
	if p.DefaultValue != auditEffect {
		t.Fatalf("expected default value %q, got %#v", auditEffect, p.DefaultValue)
	}
}

func TestBuildParameters_AllowedValues_AreMapped(t *testing.T) {
	params := []governancev1alpha1.InitiativeParameter{
		{
			Name:          "effect",
			Type:          "String",
			AllowedValues: []string{auditEffect, "Deny", "Disabled"},
		},
	}
	result := buildParameters(params)
	p := result["effect"]
	if len(p.AllowedValues) != 3 {
		t.Fatalf("expected 3 allowed values, got %d", len(p.AllowedValues))
	}
	if p.AllowedValues[0] != auditEffect {
		t.Fatalf("expected first allowed value %q, got %#v", auditEffect, p.AllowedValues[0])
	}
}

func TestBuildParameters_StrongType_SetsMetadata(t *testing.T) {
	params := []governancev1alpha1.InitiativeParameter{
		{Name: "location", Type: "String", StrongType: "location"},
	}
	result := buildParameters(params)
	p := result["location"]
	if p.Metadata == nil || p.Metadata.StrongType == nil {
		t.Fatalf("expected Metadata.StrongType to be set")
	}
	if *p.Metadata.StrongType != "location" {
		t.Fatalf("expected strong type %q, got %q", "location", *p.Metadata.StrongType)
	}
}

func TestBuildParameters_NoStrongType_NilMetadata(t *testing.T) {
	params := []governancev1alpha1.InitiativeParameter{
		{Name: "effect", Type: "String"},
	}
	result := buildParameters(params)
	if result["effect"].Metadata != nil {
		t.Fatalf("expected nil Metadata when StrongType is empty")
	}
}
