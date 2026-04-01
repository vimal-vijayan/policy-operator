package policybundle

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	annotationImportName    = "governance.platform.io/import-name"
	annotationImportMode    = "governance.platform.io/import-mode"
	importModeReconcileOnly = "reconcile"
	importModeOnlyOnce      = "adopt-once"
)

type Service struct {
	factory *client.ARMClient
}

func NewService(factory *client.ARMClient) *Service {
	return &Service{factory: factory}
}

// CreateOrUpdate creates or updates the Azure Policy Set Definition (initiative).
// resolvedPolicyDefinitionIDs contains the resolved Azure resource ID for each entry in
// spec.policyDefinitions, in the same order.
func (s *Service) CreateOrUpdate(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedPolicyDefinitionIDs []string) (string, error) {
	logger := log.FromContext(ctx)

	spec := initiative.Spec

	params := armpolicy.SetDefinition{
		Properties: &armpolicy.SetDefinitionProperties{
			DisplayName:       to.Ptr(spec.DisplayName),
			Metadata:          buildMetadata(spec.Metadata, spec.Version),
			PolicyDefinitions: buildPolicyDefinitionRefs(spec.PolicyDefinitions, resolvedPolicyDefinitionIDs),
		},
	}

	if spec.Description != "" {
		params.Properties.Description = to.Ptr(spec.Description)
	}

	if len(spec.Parameters) > 0 {
		params.Properties.Parameters = buildParameters(spec.Parameters)
	}

	initiativeName := initiative.Name

	if initiative.Annotations[annotationImportMode] == importModeReconcileOnly || initiative.Annotations[annotationImportMode] == importModeOnlyOnce {
		initiativeName = initiative.Annotations[annotationImportName]
	}

	if spec.ManagementGroupID != "" {
		logger.Info("Creating/updating policy initiative at management group scope", "name", initiativeName, "managementGroupID", spec.ManagementGroupID)
		resp, err := s.factory.Initiatives.CreateOrUpdateAtManagementGroup(ctx, spec.ManagementGroupID, initiativeName, params, nil)
		if err != nil {
			return "", err
		}
		return *resp.SetDefinition.ID, nil
	}

	logger.Info("Creating/updating policy initiative at subscription scope", "name", initiativeName)
	resp, err := s.factory.Initiatives.CreateOrUpdate(ctx, initiativeName, params, nil)
	if err != nil {
		return "", err
	}
	return *resp.SetDefinition.ID, nil
}

// Import fetches an existing Azure Policy Set Definition by its full resource ID and
// returns any drift fields between the live Azure state and the CR spec.
func (s *Service) Import(ctx context.Context, importID string, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedPolicyDefinitionIDs []string) ([]string, error) {
	logger := log.FromContext(ctx)

	if !strings.Contains(strings.ToLower(importID), "/policysetdefinitions/") {
		return nil, fmt.Errorf("import ID does not reference a Policy Set Definition: %q", importID)
	}

	initiativeName, managementGroupID, err := parseInitiativeImportID(importID)
	if err != nil {
		return nil, err
	}

	logger.Info("Fetching existing Azure Policy Set Definition for import", "importID", importID)

	props, err := s.getInitiativeProperties(ctx, importID, initiativeName, managementGroupID)
	if err != nil {
		return nil, err
	}

	if props == nil {
		return nil, fmt.Errorf("policy set definition %q returned no properties", importID)
	}

	if props.PolicyType == nil || *props.PolicyType != armpolicy.PolicyTypeCustom {
		pt := "unknown"
		if props.PolicyType != nil {
			pt = string(*props.PolicyType)
		}
		return nil, fmt.Errorf("only Custom policy set definitions can be imported, got policyType %q", pt)
	}

	var driftFields []string
	spec := initiative.Spec

	if props.DisplayName != nil && *props.DisplayName != spec.DisplayName {
		logger.V(1).Info("Drift detected in displayName", "azureValue", *props.DisplayName, "specValue", spec.DisplayName)
		driftFields = append(driftFields, "displayName")
	}
	if props.Description != nil && *props.Description != spec.Description {
		logger.V(1).Info("Drift detected in description", "azureValue", *props.Description, "specValue", spec.Description)
		driftFields = append(driftFields, "description")
	}

	if hasPolicyDefinitionDrift(props.PolicyDefinitions, resolvedPolicyDefinitionIDs) {
		logger.V(1).Info("Drift detected in policyDefinitions")
		driftFields = append(driftFields, "policyDefinitions")
	}

	return driftFields, nil
}

func (s *Service) getInitiativeProperties(ctx context.Context, importID, initiativeName, managementGroupID string) (*armpolicy.SetDefinitionProperties, error) {
	if managementGroupID != "" {
		resp, err := s.factory.Initiatives.GetAtManagementGroup(ctx, managementGroupID, initiativeName, nil)
		if err != nil {
			return nil, fmt.Errorf("fetching policy set definition %q: %w", importID, err)
		}
		return resp.SetDefinition.Properties, nil
	}

	resp, err := s.factory.Initiatives.Get(ctx, initiativeName, nil)
	if err != nil {
		return nil, fmt.Errorf("fetching policy set definition %q: %w", importID, err)
	}
	return resp.SetDefinition.Properties, nil
}

func parseInitiativeImportID(importID string) (name string, managementGroupID string, err error) {
	parts := strings.Split(importID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "policySetDefinitions") && i+1 < len(parts) {
			name = parts[i+1]
			for j, p := range parts {
				if strings.EqualFold(p, "managementGroups") && j+1 < len(parts) {
					managementGroupID = parts[j+1]
					break
				}
			}
			return name, managementGroupID, nil
		}
	}
	return "", "", fmt.Errorf("cannot parse policy set definition name from import ID: %q", importID)
}

func hasPolicyDefinitionDrift(live []*armpolicy.DefinitionReference, desired []string) bool {
	if len(live) != len(desired) {
		return true
	}

	for i := range desired {
		if live[i] == nil || live[i].PolicyDefinitionID == nil || *live[i].PolicyDefinitionID != desired[i] {
			return true
		}
	}

	return false
}

// Delete removes the Azure Policy Set Definition.
func (s *Service) Delete(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative) error {
	logger := log.FromContext(ctx)

	initiativeName := initiative.Name
	if initiative.Status.InitiativeID != "" {
		parts := strings.Split(initiative.Status.InitiativeID, "/")
		initiativeName = parts[len(parts)-1]
	}

	if initiative.Spec.ManagementGroupID != "" {
		logger.Info("Deleting policy initiative at management group scope", "name", initiativeName, "managementGroupID", initiative.Spec.ManagementGroupID)
		_, err := s.factory.Initiatives.DeleteAtManagementGroup(ctx, initiative.Spec.ManagementGroupID, initiativeName, nil)
		return err
	}

	logger.Info("Deleting policy initiative at subscription scope", "name", initiativeName)
	_, err := s.factory.Initiatives.Delete(ctx, initiativeName, nil)
	return err
}

func buildPolicyDefinitionRefs(refs []governancev1alpha1.PolicyDefinitionReference, resolvedIDs []string) []*armpolicy.DefinitionReference {
	result := make([]*armpolicy.DefinitionReference, len(refs))
	for i, ref := range refs {
		pdr := &armpolicy.DefinitionReference{
			PolicyDefinitionID: to.Ptr(resolvedIDs[i]),
		}
		if params := buildDefinitionRefParameters(ref.Parameters); len(params) > 0 {
			pdr.Parameters = params
		}
		result[i] = pdr
	}
	return result
}

func buildDefinitionRefParameters(rawRef *runtime.RawExtension) map[string]*armpolicy.ParameterValuesValue {
	if rawRef == nil {
		return nil
	}

	params := make(map[string]*armpolicy.ParameterValuesValue)
	var raw map[string]interface{}
	if err := json.Unmarshal(rawRef.Raw, &raw); err != nil {
		return nil
	}

	for k, v := range raw {
		params[k] = &armpolicy.ParameterValuesValue{Value: extractParameterValue(v)}
	}

	if len(params) == 0 {
		return nil
	}

	return params
}

func extractParameterValue(v interface{}) interface{} {
	// Each parameter entry is typically {"value": <actual>}.
	if paramMap, ok := v.(map[string]interface{}); ok {
		if val, exists := paramMap["value"]; exists {
			return val
		}
	}
	return v
}

func buildParameters(params []governancev1alpha1.InitiativeParameter) map[string]*armpolicy.ParameterDefinitionsValue {
	result := make(map[string]*armpolicy.ParameterDefinitionsValue)
	for _, p := range params {
		pv := &armpolicy.ParameterDefinitionsValue{
			Type: to.Ptr(armpolicy.ParameterType(p.Type)),
		}
		if p.DefaultValue != nil {
			var dv interface{}
			if err := json.Unmarshal(p.DefaultValue.Raw, &dv); err == nil {
				pv.DefaultValue = dv
			}
		}
		if len(p.AllowedValues) > 0 {
			av := make([]interface{}, len(p.AllowedValues))
			for i, v := range p.AllowedValues {
				av[i] = v
			}
			pv.AllowedValues = av
		}
		if p.StrongType != "" {
			pv.Metadata = &armpolicy.ParameterDefinitionsValueMetadata{
				StrongType: to.Ptr(p.StrongType),
			}
		}
		result[p.Name] = pv
	}
	return result
}

func buildMetadata(rawMeta *runtime.RawExtension, version string) interface{} {
	meta := make(map[string]interface{})

	if rawMeta != nil {
		if err := json.Unmarshal(rawMeta.Raw, &meta); err != nil {
			meta = make(map[string]interface{})
		}
	}

	if version != "" {
		meta["version"] = version
	}

	if len(meta) == 0 {
		return nil
	}
	return meta
}
