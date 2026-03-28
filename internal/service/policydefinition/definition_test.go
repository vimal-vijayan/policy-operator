package policydefinition

import (
	"context"
	"errors"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

const testDefName = "my-def"

// fakeDefinitionsAPI is a manual fake for definitions.API.
type fakeDefinitionsAPI struct {
	createOrUpdateFn            func(ctx context.Context, name string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error)
	createOrUpdateAtMgmtGroupFn func(ctx context.Context, name, mgmtGroup string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse, error)
	deleteFn                    func(ctx context.Context, name string) error
	deleteAtMgmtGroupFn         func(ctx context.Context, name, mgmtGroup string) error
}

func (f *fakeDefinitionsAPI) CreateOrUpdate(ctx context.Context, name string, params armpolicy.Definition, _ *armpolicy.DefinitionsClientCreateOrUpdateOptions) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, name, params)
	}
	return armpolicy.DefinitionsClientCreateOrUpdateResponse{}, nil
}

func (f *fakeDefinitionsAPI) CreateOrUpdateAtManagementGroup(ctx context.Context, name, mgmtGroup string, params armpolicy.Definition, _ *armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupOptions) (armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse, error) {
	if f.createOrUpdateAtMgmtGroupFn != nil {
		return f.createOrUpdateAtMgmtGroupFn(ctx, name, mgmtGroup, params)
	}
	return armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse{}, nil
}

func (f *fakeDefinitionsAPI) Delete(ctx context.Context, name string, _ *armpolicy.DefinitionsClientDeleteOptions) (armpolicy.DefinitionsClientDeleteResponse, error) {
	if f.deleteFn != nil {
		return armpolicy.DefinitionsClientDeleteResponse{}, f.deleteFn(ctx, name)
	}
	return armpolicy.DefinitionsClientDeleteResponse{}, nil
}

func (f *fakeDefinitionsAPI) DeleteAtManagementGroup(ctx context.Context, name, mgmtGroup string, _ *armpolicy.DefinitionsClientDeleteAtManagementGroupOptions) (armpolicy.DefinitionsClientDeleteAtManagementGroupResponse, error) {
	if f.deleteAtMgmtGroupFn != nil {
		return armpolicy.DefinitionsClientDeleteAtManagementGroupResponse{}, f.deleteAtMgmtGroupFn(ctx, name, mgmtGroup)
	}
	return armpolicy.DefinitionsClientDeleteAtManagementGroupResponse{}, nil
}

func (f *fakeDefinitionsAPI) Get(_ context.Context, _ string, _ *armpolicy.DefinitionsClientGetOptions) (armpolicy.DefinitionsClientGetResponse, error) {
	return armpolicy.DefinitionsClientGetResponse{}, nil
}

func (f *fakeDefinitionsAPI) GetAtManagementGroup(_ context.Context, _, _ string, _ *armpolicy.DefinitionsClientGetAtManagementGroupOptions) (armpolicy.DefinitionsClientGetAtManagementGroupResponse, error) {
	return armpolicy.DefinitionsClientGetAtManagementGroupResponse{}, nil
}

// helpers

func newTestDefinitionService(api *fakeDefinitionsAPI) *Service {
	return NewService(&client.ARMClient{Definitions: api})
}

func newDefinition(name string, spec governancev1alpha1.AzurePolicyDefinitionSpec) *governancev1alpha1.AzurePolicyDefinition {
	return &governancev1alpha1.AzurePolicyDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       spec,
	}
}

func ptr[T any](v T) *T { return &v }

// ── CreateOrUpdate ──────────────────────────────────────────────────────────

func TestCreateOrUpdate_SubscriptionScope_ReturnsID(t *testing.T) {
	const fakeID = "/subscriptions/sub1/providers/Microsoft.Authorization/policyDefinitions/my-def"
	ctx := context.Background()

	var gotName string
	var gotParams armpolicy.Definition

	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, name string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			gotName = name
			gotParams = params
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{
				Definition: armpolicy.Definition{ID: ptr(fakeID)},
			}, nil
		},
	}

	def := newDefinition(testDefName, governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName: "My Definition",
		Mode:        "All",
	})

	id, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != fakeID {
		t.Fatalf("expected ID %q, got %q", fakeID, id)
	}
	if gotName != testDefName {
		t.Fatalf("expected name %q, got %q", testDefName, gotName)
	}
	if gotParams.Properties == nil || *gotParams.Properties.DisplayName != "My Definition" {
		t.Fatalf("unexpected display name in params: %#v", gotParams.Properties)
	}
	if *gotParams.Properties.Mode != "All" {
		t.Fatalf("expected mode %q, got %q", "All", *gotParams.Properties.Mode)
	}
}

func TestCreateOrUpdate_SubscriptionScope_SetsDescription(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Definition
	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{
				Definition: armpolicy.Definition{ID: ptr("id")},
			}, nil
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName: "Def",
		Mode:        "All",
		Description: "a description",
	})

	if _, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.Description == nil || *gotParams.Properties.Description != "a description" {
		t.Fatalf("expected description to be set, got %v", gotParams.Properties.Description)
	}
}

func TestCreateOrUpdate_SubscriptionScope_SetsPolicyType(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Definition
	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{
				Definition: armpolicy.Definition{ID: ptr("id")},
			}, nil
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName: "Def",
		Mode:        "All",
		PolicyType:  "Custom",
	})

	if _, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.PolicyType == nil || *gotParams.Properties.PolicyType != armpolicy.PolicyTypeCustom {
		t.Fatalf("expected PolicyType Custom, got %v", gotParams.Properties.PolicyType)
	}
}

func TestCreateOrUpdate_SubscriptionScope_ParsesParameters(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Definition
	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{
				Definition: armpolicy.Definition{ID: ptr("id")},
			}, nil
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName: "Def",
		Mode:        "All",
		Parameters:  &runtime.RawExtension{Raw: []byte(`{"effect":{"type":"String","defaultValue":"Deny"}}`)},
	})

	if _, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(gotParams.Properties.Parameters) != 1 {
		t.Fatalf("expected 1 parameter, got %d", len(gotParams.Properties.Parameters))
	}
	if _, ok := gotParams.Properties.Parameters["effect"]; !ok {
		t.Fatalf("expected 'effect' parameter to be present")
	}
}

func TestCreateOrUpdate_SubscriptionScope_ParsesPolicyRule(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Definition
	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{
				Definition: armpolicy.Definition{ID: ptr("id")},
			}, nil
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName: "Def",
		Mode:        "All",
		PolicyRule:  &runtime.RawExtension{Raw: []byte(`{"if":{"field":"type","equals":"Microsoft.Compute/virtualMachines"},"then":{"effect":"deny"}}`)},
	})

	if _, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.PolicyRule == nil {
		t.Fatal("expected PolicyRule to be set")
	}
}

func TestCreateOrUpdate_SubscriptionScope_ParsesPolicyRuleJSON(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Definition
	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{
				Definition: armpolicy.Definition{ID: ptr("id")},
			}, nil
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName:    "Def",
		Mode:           "All",
		PolicyRuleJSON: `{"if":{"field":"type","equals":"Microsoft.Compute/virtualMachines"},"then":{"effect":"deny"}}`,
	})

	if _, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotParams.Properties.PolicyRule == nil {
		t.Fatal("expected PolicyRule to be set via PolicyRuleJSON")
	}
}

func TestCreateOrUpdate_SubscriptionScope_PolicyRuleTakesPrecedenceOverJSON(t *testing.T) {
	ctx := context.Background()

	var gotParams armpolicy.Definition
	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, _ string, params armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			gotParams = params
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{
				Definition: armpolicy.Definition{ID: ptr("id")},
			}, nil
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName:    "Def",
		Mode:           "All",
		PolicyRule:     &runtime.RawExtension{Raw: []byte(`{"source":"raw"}`)},
		PolicyRuleJSON: `{"source":"json"}`,
	})

	if _, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	rule, ok := gotParams.Properties.PolicyRule.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected PolicyRule type: %T", gotParams.Properties.PolicyRule)
	}
	if rule["source"] != "raw" {
		t.Fatalf("expected PolicyRule from raw extension, got source=%v", rule["source"])
	}
}

func TestCreateOrUpdate_ManagementGroupScope_CallsManagementGroupAPI(t *testing.T) {
	const fakeID = "/providers/Microsoft.Management/managementGroups/mg1/providers/Microsoft.Authorization/policyDefinitions/my-def"
	ctx := context.Background()

	var gotMgmtGroup string
	api := &fakeDefinitionsAPI{
		createOrUpdateAtMgmtGroupFn: func(_ context.Context, _, mgmtGroup string, _ armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse, error) {
			gotMgmtGroup = mgmtGroup
			return armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse{
				Definition: armpolicy.Definition{ID: ptr(fakeID)},
			}, nil
		},
	}

	def := newDefinition(testDefName, governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName:       "My Definition",
		Mode:              "All",
		ManagementGroupID: "mg1",
	})

	id, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if id != fakeID {
		t.Fatalf("expected ID %q, got %q", fakeID, id)
	}
	if gotMgmtGroup != "mg1" {
		t.Fatalf("expected management group %q, got %q", "mg1", gotMgmtGroup)
	}
}

func TestCreateOrUpdate_SubscriptionScope_PropagatesError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("azure api error")

	api := &fakeDefinitionsAPI{
		createOrUpdateFn: func(_ context.Context, _ string, _ armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateResponse, error) {
			return armpolicy.DefinitionsClientCreateOrUpdateResponse{}, expectedErr
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName: "Def",
		Mode:        "All",
	})

	_, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestCreateOrUpdate_ManagementGroupScope_PropagatesError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("mgmt group api error")

	api := &fakeDefinitionsAPI{
		createOrUpdateAtMgmtGroupFn: func(_ context.Context, _, _ string, _ armpolicy.Definition) (armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse, error) {
			return armpolicy.DefinitionsClientCreateOrUpdateAtManagementGroupResponse{}, expectedErr
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		DisplayName:       "Def",
		Mode:              "All",
		ManagementGroupID: "mg1",
	})

	_, err := newTestDefinitionService(api).CreateOrUpdate(ctx, def)
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

// ── Delete ──────────────────────────────────────────────────────────────────

func TestDelete_SubscriptionScope_CallsDeleteWithName(t *testing.T) {
	ctx := context.Background()

	var gotName string
	api := &fakeDefinitionsAPI{
		deleteFn: func(_ context.Context, name string) error {
			gotName = name
			return nil
		},
	}

	def := newDefinition(testDefName, governancev1alpha1.AzurePolicyDefinitionSpec{})

	if err := newTestDefinitionService(api).Delete(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != testDefName {
		t.Fatalf("expected name %q, got %q", testDefName, gotName)
	}
}

func TestDelete_SubscriptionScope_ExtractsNameFromStatusID(t *testing.T) {
	ctx := context.Background()

	var gotName string
	api := &fakeDefinitionsAPI{
		deleteFn: func(_ context.Context, name string) error {
			gotName = name
			return nil
		},
	}

	def := newDefinition("original-name", governancev1alpha1.AzurePolicyDefinitionSpec{})
	def.Status.PolicyDefinitionID = "/subscriptions/sub1/providers/Microsoft.Authorization/policyDefinitions/extracted-name"

	if err := newTestDefinitionService(api).Delete(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != "extracted-name" {
		t.Fatalf("expected name %q, got %q", "extracted-name", gotName)
	}
}

func TestDelete_ManagementGroupScope_CallsDeleteAtManagementGroup(t *testing.T) {
	ctx := context.Background()

	var gotName, gotMgmtGroup string
	api := &fakeDefinitionsAPI{
		deleteAtMgmtGroupFn: func(_ context.Context, name, mgmtGroup string) error {
			gotName = name
			gotMgmtGroup = mgmtGroup
			return nil
		},
	}

	def := newDefinition(testDefName, governancev1alpha1.AzurePolicyDefinitionSpec{
		ManagementGroupID: "mg1",
	})

	if err := newTestDefinitionService(api).Delete(ctx, def); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotName != testDefName {
		t.Fatalf("expected name %q, got %q", testDefName, gotName)
	}
	if gotMgmtGroup != "mg1" {
		t.Fatalf("expected management group %q, got %q", "mg1", gotMgmtGroup)
	}
}

func TestDelete_SubscriptionScope_PropagatesError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("delete error")

	api := &fakeDefinitionsAPI{
		deleteFn: func(_ context.Context, _ string) error {
			return expectedErr
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{})

	if err := newTestDefinitionService(api).Delete(ctx, def); !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

func TestDelete_ManagementGroupScope_PropagatesError(t *testing.T) {
	ctx := context.Background()
	expectedErr := errors.New("mgmt delete error")

	api := &fakeDefinitionsAPI{
		deleteAtMgmtGroupFn: func(_ context.Context, _, _ string) error {
			return expectedErr
		},
	}

	def := newDefinition("def", governancev1alpha1.AzurePolicyDefinitionSpec{
		ManagementGroupID: "mg1",
	})

	if err := newTestDefinitionService(api).Delete(ctx, def); !errors.Is(err, expectedErr) {
		t.Fatalf("expected error %v, got %v", expectedErr, err)
	}
}

// ── buildMetadata ────────────────────────────────────────────────────────────

func TestBuildMetadata_NilInputsReturnsNil(t *testing.T) {
	result := buildMetadata(nil, "")
	if result != nil {
		t.Fatalf("expected nil, got %v", result)
	}
}

func TestBuildMetadata_VersionOnlySetsVersionKey(t *testing.T) {
	result := buildMetadata(nil, "1.2.3")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	if m["version"] != "1.2.3" {
		t.Fatalf("expected version %q, got %v", "1.2.3", m["version"])
	}
}

func TestBuildMetadata_RawMetaOnlyPreservesKeys(t *testing.T) {
	raw := &runtime.RawExtension{Raw: []byte(`{"category":"Security"}`)}
	result := buildMetadata(raw, "")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	if m["category"] != "Security" {
		t.Fatalf("expected category %q, got %v", "Security", m["category"])
	}
}

func TestBuildMetadata_VersionOverridesRawMetaVersion(t *testing.T) {
	raw := &runtime.RawExtension{Raw: []byte(`{"version":"0.0.1","category":"Security"}`)}
	result := buildMetadata(raw, "2.0.0")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	if m["version"] != "2.0.0" {
		t.Fatalf("spec.Version should override raw version: got %v", m["version"])
	}
	if m["category"] != "Security" {
		t.Fatalf("expected category to be preserved, got %v", m["category"])
	}
}

func TestBuildMetadata_InvalidJSONFallsBackToVersionOnly(t *testing.T) {
	raw := &runtime.RawExtension{Raw: []byte(`not-valid-json`)}
	result := buildMetadata(raw, "1.0.0")
	m, ok := result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected map[string]interface{}, got %T", result)
	}
	if m["version"] != "1.0.0" {
		t.Fatalf("expected version %q after invalid JSON, got %v", "1.0.0", m["version"])
	}
}
