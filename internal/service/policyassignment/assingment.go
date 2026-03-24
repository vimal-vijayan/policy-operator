package policyassignment

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/google/uuid"
	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Service struct {
	factory *client.ARMClient
}

func NewService(factory *client.ARMClient) *Service {
	return &Service{
		factory: factory,
	}
}

func (s *Service) CreateOrUpdate(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, error) {
	logger := log.FromContext(ctx)

	// Use stable name from existing assignment ID, or generate a new UUID
	assignmentName := ""
	if assignment.Status.AssignmentID != "" {
		parts := strings.Split(assignment.Status.AssignmentID, "/")
		assignmentName = parts[len(parts)-1]
	} else {
		assignmentName = uuid.NewString()
	}

	spec := assignment.Spec

	params := armpolicy.Assignment{
		Properties: &armpolicy.AssignmentProperties{
			DisplayName:        to.Ptr(spec.DisplayName),
			PolicyDefinitionID: to.Ptr(policyDefinitionID),
		},
	}

	if spec.Description != "" {
		params.Properties.Description = to.Ptr(spec.Description)
	}

	if len(spec.NotScopes) > 0 {
		notScopes := make([]*string, len(spec.NotScopes))
		for i, ns := range spec.NotScopes {
			notScopes[i] = to.Ptr(ns)
		}
		params.Properties.NotScopes = notScopes
	}

	if spec.EnforcementMode != "" {
		params.Properties.EnforcementMode = to.Ptr(armpolicy.EnforcementMode(spec.EnforcementMode))
	}

	if spec.Parameters != nil {
		var paramValues map[string]*armpolicy.ParameterValuesValue
		if err := json.Unmarshal(spec.Parameters.Raw, &paramValues); err == nil {
			params.Properties.Parameters = paramValues
		} else {
			var flat map[string]interface{}
			if err := json.Unmarshal(spec.Parameters.Raw, &flat); err == nil {
				wrapped := make(map[string]*armpolicy.ParameterValuesValue, len(flat))
				for k, v := range flat {
					wrapped[k] = &armpolicy.ParameterValuesValue{Value: v}
				}
				params.Properties.Parameters = wrapped
			}
		}
	}

	if spec.Metadata != nil {
		var meta interface{}
		if err := json.Unmarshal(spec.Metadata.Raw, &meta); err == nil {
			params.Properties.Metadata = meta
		}
	}

	if spec.Identity != nil {
		params.Identity = &armpolicy.Identity{
			Type: to.Ptr(armpolicy.ResourceIdentityType(spec.Identity.Type)),
		}
		if spec.Identity.Type == "UserAssigned" && spec.Identity.UserAssignedIdentityID != "" {
			params.Identity.UserAssignedIdentities = map[string]*armpolicy.UserAssignedIdentitiesValue{
				spec.Identity.UserAssignedIdentityID: {},
			}
		}
	}

	logger.Info("Creating or updating Azure Policy Assignment", "name", assignmentName, "scope", spec.Scope)

	resp, err := s.factory.Assignments.Create(ctx, spec.Scope, assignmentName, params, nil)
	if err != nil {
		return "", err
	}

	return *resp.Assignment.ID, nil
}

func (s *Service) Delete(ctx context.Context, scope string, assignmentID string) error {
	logger := log.FromContext(ctx)

	parts := strings.Split(assignmentID, "/")
	assignmentName := parts[len(parts)-1]

	logger.Info("Deleting Azure Policy Assignment", "name", assignmentName, "scope", scope)

	_, err := s.factory.Assignments.Delete(ctx, scope, assignmentName, nil)
	return err
}
