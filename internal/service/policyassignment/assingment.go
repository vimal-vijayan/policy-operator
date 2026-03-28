package policyassignment

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
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

//nolint:gocyclo
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

	if spec.NonComplianceMessages != nil {
		messages := make([]*armpolicy.NonComplianceMessage, 0, 1+len(spec.NonComplianceMessages.PerPolicy))

		if spec.NonComplianceMessages.Default != "" {
			messages = append(messages, &armpolicy.NonComplianceMessage{
				Message: to.Ptr(spec.NonComplianceMessages.Default),
			})
		}

		for _, perPolicy := range spec.NonComplianceMessages.PerPolicy {
			if perPolicy.Message == "" {
				continue
			}
			messages = append(messages, &armpolicy.NonComplianceMessage{
				Message:                     to.Ptr(perPolicy.Message),
				PolicyDefinitionReferenceID: to.Ptr(perPolicy.PolicyReferenceID),
			})
		}

		if len(messages) > 0 {
			params.Properties.NonComplianceMessages = messages
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

	if len(spec.ResourceSelectors) > 0 {
		resourceSelectors := make([]*armpolicy.ResourceSelector, len(spec.ResourceSelectors))
		for i, rs := range spec.ResourceSelectors {
			selectors := make([]*armpolicy.Selector, len(rs.Selectors))
			for j, sel := range rs.Selectors {
				s := &armpolicy.Selector{
					Kind: to.Ptr(armpolicy.SelectorKind(sel.Property)),
				}
				vals := make([]*string, len(sel.Values))
				for k, v := range sel.Values {
					vals[k] = to.Ptr(v)
				}
				switch sel.Operator {
				case "In":
					s.In = vals
				case "notIn":
					s.NotIn = vals
				}
				selectors[j] = s
			}
			resourceSelectors[i] = &armpolicy.ResourceSelector{
				Name:      to.Ptr(rs.Name),
				Selectors: selectors,
			}
		}
		params.Properties.ResourceSelectors = resourceSelectors
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

	miPrincipalID := ""
	if resp.Assignment.Identity != nil && resp.Assignment.Identity.PrincipalID != nil {
		miPrincipalID = *resp.Assignment.Identity.PrincipalID
	}

	assignedLocation := ""
	if resp.Assignment.Location != nil {
		assignedLocation = *resp.Assignment.Location
	}

	exemptionStatuses, err := s.reconcileExemptions(ctx, assignment, assignmentID)
	if err != nil {
		return assignmentID, assignedLocation, miPrincipalID, nil, err
	}

	if err := s.reconcileRoleAssignments(ctx, assignment, assignmentID, miPrincipalID); err != nil {
		return assignmentID, assignedLocation, miPrincipalID, exemptionStatuses, err
	}

	return assignmentID, assignedLocation, miPrincipalID, exemptionStatuses, nil
}

// reconcileRoleAssignments creates role assignments for the managed identity based on spec permissions.
// It is a no-op if the identity is nil, has no permissions, or no principal ID was returned.
func (s *Service) reconcileRoleAssignments(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, assignmentID, principalID string) error {
	logger := log.FromContext(ctx)

	spec := assignment.Spec
	if spec.Identity == nil || len(spec.Identity.Permissions) == 0 || principalID == "" {
		return nil
	}

	// Deterministic namespace UUID for generating stable role assignment names.
	raNamespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

	for _, perm := range spec.Identity.Permissions {
		roleDefinitionID := perm.RoleDefinitionID

		if roleDefinitionID == "" {
			if perm.Role == "" {
				return fmt.Errorf("identity permission at scope %q: either role or roleDefinitionId must be specified", perm.Scope)
			}
			resolved, err := s.resolveRoleDefinitionID(ctx, perm.Scope, perm.Role)
			if err != nil {
				return fmt.Errorf("resolving role %q at scope %q: %w", perm.Role, perm.Scope, err)
			}
			roleDefinitionID = resolved
		}

		// Stable name: deterministic UUID derived from assignment ID + scope + role definition ID.
		raName := uuid.NewSHA1(raNamespace, []byte(assignmentID+perm.Scope+roleDefinitionID)).String()

		logger.Info("Creating role assignment for managed identity",
			"principalId", principalID,
			"roleDefinitionId", roleDefinitionID,
			"scope", perm.Scope,
			"roleAssignmentName", raName,
		)

		if _, err := s.factory.RoleAssignments.Create(ctx, perm.Scope, raName, armauthorization.RoleAssignmentCreateParameters{
			Properties: &armauthorization.RoleAssignmentProperties{
				PrincipalID:      to.Ptr(principalID),
				RoleDefinitionID: to.Ptr(roleDefinitionID),
				PrincipalType:    to.Ptr(armauthorization.PrincipalTypeServicePrincipal),
			},
		}, nil); err != nil {
			return fmt.Errorf("creating role assignment at scope %q: %w", perm.Scope, err)
		}
	}

	return nil
}

// resolveRoleDefinitionID looks up the Azure role definition ID for a given role name at the provided scope.
func (s *Service) resolveRoleDefinitionID(ctx context.Context, scope, roleName string) (string, error) {
	filter := fmt.Sprintf("roleName eq '%s'", roleName)
	pager := s.factory.RoleDefinitions.NewListPager(scope, &armauthorization.RoleDefinitionsClientListOptions{
		Filter: to.Ptr(filter),
	})

	for pager.More() {
		page, err := pager.NextPage(ctx)
		if err != nil {
			return "", err
		}
		for _, rd := range page.Value {
			if rd.ID != nil {
				return *rd.ID, nil
			}
		}
	}

	return "", fmt.Errorf("role %q not found at scope %q", roleName, scope)
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
	results := make([]governancev1alpha1.AssignmentExemptionStatus, 0)

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

func (s *Service) Delete(ctx context.Context, scope string, assignmentID string, exemptions []governancev1alpha1.AssignmentExemptionStatus, identity *governancev1alpha1.AssignmentIdentity) error {
	logger := log.FromContext(ctx)

	// Delete all inline exemptions before removing the assignment.
	for _, e := range exemptions {
		logger.Info("Deleting inline exemption", "displayName", e.DisplayName, "exemptionId", e.ExemptionID)
		if err := s.exemptionService.Delete(ctx, e.Scope, e.ExemptionID); err != nil {
			return fmt.Errorf("deleting exemption %q: %w", e.DisplayName, err)
		}
	}

	// Delete role assignments created for the managed identity, re-deriving names from spec.
	if identity != nil && len(identity.Permissions) > 0 {
		raNamespace := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
		for _, perm := range identity.Permissions {
			roleDefinitionID := perm.RoleDefinitionID
			if roleDefinitionID == "" {
				resolved, err := s.resolveRoleDefinitionID(ctx, perm.Scope, perm.Role)
				if err != nil {
					return fmt.Errorf("resolving role %q at scope %q for deletion: %w", perm.Role, perm.Scope, err)
				}
				roleDefinitionID = resolved
			}
			raName := uuid.NewSHA1(raNamespace, []byte(assignmentID+perm.Scope+roleDefinitionID)).String()
			logger.Info("Deleting role assignment", "scope", perm.Scope, "roleAssignmentName", raName)
			if _, err := s.factory.RoleAssignments.Delete(ctx, perm.Scope, raName, nil); err != nil {
				return fmt.Errorf("deleting role assignment at scope %q: %w", perm.Scope, err)
			}
		}
	}

	parts := strings.Split(assignmentID, "/")
	assignmentName := parts[len(parts)-1]

	logger.Info("Deleting Azure Policy Assignment", "name", assignmentName, "scope", scope)

	_, err := s.factory.Assignments.Delete(ctx, scope, assignmentName, nil)
	return err
}
