package policyassignment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/google/uuid"
	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/client"
	"github.com/vimal-vijayan/azure-policy-operator/internal/service/policyexemption"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Service struct {
	factory          *client.ARMClient
	exemptionService *policyexemption.Service
}

func NewService(factory *client.ARMClient, exemptionService *policyexemption.Service) *Service {
	return &Service{
		factory:          factory,
		exemptionService: exemptionService,
	}
}

func (s *Service) CreateOrUpdate(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
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

	if spec.NonComplianceMessage != "" {
		params.Properties.NonComplianceMessages = []*armpolicy.NonComplianceMessage{
			{
				Message: to.Ptr(spec.NonComplianceMessage),
			},
		}
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

	if spec.Identity != nil && spec.Identity.Type != "None" {
		logger.Info("Configuring managed identity for policy assignment", "type", spec.Identity.Type, "location", spec.Identity.Location, "userAssignedIdentityId", spec.Identity.UserAssignedIdentityID)
		params.Identity = &armpolicy.Identity{
			Type: to.Ptr(armpolicy.ResourceIdentityType(spec.Identity.Type)),
		}
		if spec.Identity.Location != "" {
			params.Location = to.Ptr(spec.Identity.Location)
		}
		if spec.Identity.Type == "UserAssigned" && spec.Identity.UserAssignedIdentityID != "" {
			params.Identity.UserAssignedIdentities = map[string]*armpolicy.UserAssignedIdentitiesValue{
				spec.Identity.UserAssignedIdentityID: {},
			}
		}
	} else if assignment.Status.AssignedLocation != "" {
		// Preserve the existing location on updates — Azure rejects changing it to empty.
		params.Location = to.Ptr(assignment.Status.AssignedLocation)
	}

	logger.Info("Creating or updating Azure Policy Assignment", "name", assignmentName, "scope", spec.Scope)

	resp, err := s.factory.Assignments.Create(ctx, spec.Scope, assignmentName, params, nil)
	if err != nil {
		return "", "", "", nil, err
	}

	assignmentID := *resp.Assignment.ID
	miPrinicpalID := *resp.Assignment.Identity.PrincipalID

	assignedLocation := ""
	if resp.Assignment.Location != nil {
		assignedLocation = *resp.Assignment.Location
	}

	exemptionStatuses, err := s.reconcileExemptions(ctx, assignment, assignmentID)
	if err != nil {
		return assignmentID, assignedLocation, miPrinicpalID, nil, err
	}

	return assignmentID, assignedLocation, miPrinicpalID, exemptionStatuses, nil
}

// reconcileExemptions creates/updates exemptions present in the spec and deletes ones removed from it.
func (s *Service) reconcileExemptions(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, assignmentID string) ([]governancev1alpha1.AssignmentExemptionStatus, error) {
	logger := log.FromContext(ctx)

	// Build lookup from existing status: displayName -> AssignmentExemptionStatus
	existing := make(map[string]governancev1alpha1.AssignmentExemptionStatus)
	for _, e := range assignment.Status.Exemptions {
		existing[e.DisplayName] = e
	}

	desired := make(map[string]bool)
	var results []governancev1alpha1.AssignmentExemptionStatus

	for _, exemptionSpec := range assignment.Spec.Exemptions {
		desired[exemptionSpec.DisplayName] = true

		// Construct a synthetic AzurePolicyExemption to reuse the exemption service logic
		synthetic := &governancev1alpha1.AzurePolicyExemption{
			Spec: governancev1alpha1.AzurePolicyExemptionSpec{
				DisplayName:        exemptionSpec.DisplayName,
				Description:        exemptionSpec.Description,
				PolicyAssignmentID: assignmentID,
				Scope:              exemptionSpec.Scope,
				ExemptionCategory:  exemptionSpec.ExemptionCategory,
				ExpiresOn:          exemptionSpec.ExpiresOn,
			},
		}

		// Restore the existing exemption ID for stable naming on updates
		if prev, ok := existing[exemptionSpec.DisplayName]; ok {
			synthetic.Status.ExemptionID = prev.ExemptionID
		}

		exemptionID, err := s.exemptionService.CreateOrUpdate(ctx, synthetic)
		if err != nil {
			return nil, fmt.Errorf("reconciling exemption %q: %w", exemptionSpec.DisplayName, err)
		}

		results = append(results, governancev1alpha1.AssignmentExemptionStatus{
			DisplayName: exemptionSpec.DisplayName,
			ExemptionID: exemptionID,
			Scope:       exemptionSpec.Scope,
		})
	}

	// Delete exemptions that were removed from the spec
	for _, prev := range assignment.Status.Exemptions {
		if !desired[prev.DisplayName] {
			logger.Info("Deleting removed exemption", "displayName", prev.DisplayName, "exemptionId", prev.ExemptionID)
			if err := s.exemptionService.Delete(ctx, prev.Scope, prev.ExemptionID); err != nil {
				return nil, fmt.Errorf("deleting exemption %q: %w", prev.DisplayName, err)
			}
		}
	}

	return results, nil
}

func (s *Service) Delete(ctx context.Context, scope string, assignmentID string, exemptions []governancev1alpha1.AssignmentExemptionStatus) error {
	logger := log.FromContext(ctx)

	// Delete all inline exemptions before removing the assignment
	for _, e := range exemptions {
		logger.Info("Deleting inline exemption", "displayName", e.DisplayName, "exemptionId", e.ExemptionID)
		if err := s.exemptionService.Delete(ctx, e.Scope, e.ExemptionID); err != nil {
			return fmt.Errorf("deleting exemption %q: %w", e.DisplayName, err)
		}
	}

	parts := strings.Split(assignmentID, "/")
	assignmentName := parts[len(parts)-1]

	logger.Info("Deleting Azure Policy Assignment", "name", assignmentName, "scope", scope)

	_, err := s.factory.Assignments.Delete(ctx, scope, assignmentName, nil)
	return err
}
