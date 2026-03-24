package policydefinition

import (
	"context"
	"encoding/json"
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
