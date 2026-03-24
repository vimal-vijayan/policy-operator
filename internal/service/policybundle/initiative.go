package policybundle

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

	if spec.ManagementGroupID != "" {
		logger.Info("Creating/updating policy initiative at management group scope", "name", initiativeName, "managementGroupID", spec.ManagementGroupID)
		resp, err := s.factory.Initiatives.CreateOrUpdateAtManagementGroup(ctx, initiativeName, spec.ManagementGroupID, params, nil)
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
		_, err := s.factory.Initiatives.DeleteAtManagementGroup(ctx, initiativeName, initiative.Spec.ManagementGroupID, nil)
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
		if ref.Parameters != nil {
			params := make(map[string]*armpolicy.ParameterValuesValue)
			var raw map[string]interface{}
			if err := json.Unmarshal(ref.Parameters.Raw, &raw); err == nil {
				for k, v := range raw {
					// Each parameter entry is {"value": <actual>} — extract the inner value.
					if paramMap, ok := v.(map[string]interface{}); ok {
						if val, exists := paramMap["value"]; exists {
							params[k] = &armpolicy.ParameterValuesValue{Value: val}
							continue
						}
					}
					params[k] = &armpolicy.ParameterValuesValue{Value: v}
				}
			}
			if len(params) > 0 {
				pdr.Parameters = params
			}
		}
		result[i] = pdr
	}
	return result
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
