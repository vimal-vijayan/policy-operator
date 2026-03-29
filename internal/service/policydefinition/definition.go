package policydefinition

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

type Service struct {
	factory *client.ARMClient
}

func NewService(factory *client.ARMClient) *Service {
	return &Service{factory: factory}
}

func (s *Service) CreateOrUpdate(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (string, error) {
	logger := log.FromContext(ctx)

	spec := def.Spec
	params := armpolicy.Definition{
		Properties: &armpolicy.DefinitionProperties{
			DisplayName: to.Ptr(spec.DisplayName),
			Mode:        to.Ptr(spec.Mode),
			Metadata:    buildMetadata(spec.Metadata, spec.Version),
		},
	}

	if spec.Description != "" {
		params.Properties.Description = to.Ptr(spec.Description)
	}

	if spec.PolicyType != "" {
		params.Properties.PolicyType = to.Ptr(armpolicy.PolicyType(spec.PolicyType))
	}

	if spec.Parameters != nil {
		var paramDefs map[string]*armpolicy.ParameterDefinitionsValue
		if err := json.Unmarshal(spec.Parameters.Raw, &paramDefs); err == nil {
			params.Properties.Parameters = paramDefs
		}
	}

	if spec.PolicyRule != nil {
		var rule interface{}
		if err := json.Unmarshal(spec.PolicyRule.Raw, &rule); err == nil {
			params.Properties.PolicyRule = rule
		}
	} else if spec.PolicyRuleJSON != "" {
		var rule interface{}
		if err := json.Unmarshal([]byte(spec.PolicyRuleJSON), &rule); err == nil {
			params.Properties.PolicyRule = rule
		}
	}

	policyName := def.Name

	if spec.ManagementGroupID != "" {
		logger.Info("Creating/updating policy definition at management group scope", "name", policyName, "managementGroupID", spec.ManagementGroupID)
		resp, err := s.factory.Definitions.CreateOrUpdateAtManagementGroup(ctx, policyName, spec.ManagementGroupID, params, nil)
		if err != nil {
			return "", err
		}
		return *resp.Definition.ID, nil
	}

	logger.Info("Creating/updating policy definition at subscription scope", "name", policyName)
	resp, err := s.factory.Definitions.CreateOrUpdate(ctx, policyName, params, nil)
	if err != nil {
		return "", err
	}
	return *resp.Definition.ID, nil
}

// Import fetches an existing Azure Policy Definition by its full resource ID, validates that it is
// of type Custom, and returns any drift fields between the live Azure state and the CR spec.
func (s *Service) Import(ctx context.Context, importID string, def *governancev1alpha1.AzurePolicyDefinition) ([]string, error) {
	logger := log.FromContext(ctx)

	if !strings.Contains(strings.ToLower(importID), "/policydefinitions/") {
		return nil, fmt.Errorf("import ID does not reference a Policy Definition: %q", importID)
	}

	defName, managementGroupID, err := parseDefinitionImportID(importID)
	if err != nil {
		return nil, err
	}

	logger.Info("Fetching existing Azure Policy Definition for import", "importID", importID)

	var props *armpolicy.DefinitionProperties
	if managementGroupID != "" {
		resp, err := s.factory.Definitions.GetAtManagementGroup(ctx, defName, managementGroupID, nil)
		if err != nil {
			return nil, fmt.Errorf("fetching policy definition %q: %w", importID, err)
		}
		props = resp.Definition.Properties
	} else {
		resp, err := s.factory.Definitions.Get(ctx, defName, nil)
		if err != nil {
			return nil, fmt.Errorf("fetching policy definition %q: %w", importID, err)
		}
		props = resp.Definition.Properties
	}

	if props == nil {
		return nil, fmt.Errorf("policy definition %q returned no properties", importID)
	}

	if props.PolicyType == nil || *props.PolicyType != armpolicy.PolicyTypeCustom {
		pt := "unknown"
		if props.PolicyType != nil {
			pt = string(*props.PolicyType)
		}
		return nil, fmt.Errorf("only Custom policy definitions can be imported, got policyType %q", pt)
	}

	var driftFields []string
	spec := def.Spec

	if props.DisplayName != nil && *props.DisplayName != spec.DisplayName {
		logger.V(1).Info("Drift detected in displayName", "azureValue", *props.DisplayName, "specValue", spec.DisplayName)
		driftFields = append(driftFields, "displayName")
	}
	if props.Description != nil && *props.Description != spec.Description {
		logger.V(1).Info("Drift detected in description", "azureValue", *props.Description, "specValue", spec.Description)
		driftFields = append(driftFields, "description")
	}
	if props.Mode != nil && *props.Mode != spec.Mode {
		logger.V(1).Info("Drift detected in mode", "azureValue", *props.Mode, "specValue", spec.Mode)
		driftFields = append(driftFields, "mode")
	}

	return driftFields, nil
}

func parseDefinitionImportID(importID string) (name string, managementGroupID string, err error) {
	parts := strings.Split(importID, "/")
	for i, part := range parts {
		if strings.EqualFold(part, "policyDefinitions") && i+1 < len(parts) {
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
	return "", "", fmt.Errorf("cannot parse policy definition name from import ID: %q", importID)
}

func (s *Service) Delete(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) error {
	logger := log.FromContext(ctx)

	policyName := def.Name
	if def.Status.PolicyDefinitionID != "" {
		parts := strings.Split(def.Status.PolicyDefinitionID, "/")
		policyName = parts[len(parts)-1]
	}

	if def.Spec.ManagementGroupID != "" {
		logger.Info("Deleting policy definition at management group scope", "name", policyName, "managementGroupID", def.Spec.ManagementGroupID)
		_, err := s.factory.Definitions.DeleteAtManagementGroup(ctx, policyName, def.Spec.ManagementGroupID, nil)
		return err
	}

	logger.Info("Deleting policy definition at subscription scope", "name", policyName)
	_, err := s.factory.Definitions.Delete(ctx, policyName, nil)
	return err
}

// buildMetadata merges spec.Version into the metadata map under the key "version".
// spec.Version always takes precedence over any "version" key in rawMeta.
// Returns nil when both inputs are empty/nil.
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
