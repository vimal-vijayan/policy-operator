package policyexemption

import (
	"context"
	"strings"
	"time"

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
	return &Service{factory: factory}
}

func (s *Service) CreateOrUpdate(ctx context.Context, exemption *governancev1alpha1.AzurePolicyExemption) (string, error) {
	logger := log.FromContext(ctx)

	exemptionName := ""
	if exemption.Status.ExemptionID != "" {
		parts := strings.Split(exemption.Status.ExemptionID, "/")
		exemptionName = parts[len(parts)-1]
	} else {
		exemptionName = uuid.NewString()
	}

	spec := exemption.Spec

	params := armpolicy.Exemption{
		Properties: &armpolicy.ExemptionProperties{
			DisplayName:        to.Ptr(spec.DisplayName),
			PolicyAssignmentID: to.Ptr(spec.PolicyAssignmentID),
			ExemptionCategory:  to.Ptr(armpolicy.ExemptionCategory(spec.ExemptionCategory)),
		},
	}

	if spec.Description != "" {
		params.Properties.Description = to.Ptr(spec.Description)
	}

	if spec.ExpiresOn != "" {
		t, err := time.Parse(time.RFC3339, spec.ExpiresOn)
		if err == nil {
			params.Properties.ExpiresOn = to.Ptr(t)
		}
	}

	if spec.ResourceSelector != nil {
		params.Properties.ResourceSelectors = []*armpolicy.ResourceSelector{
			{
				Selectors: []*armpolicy.Selector{
					{
						Kind: to.Ptr(armpolicy.SelectorKind(spec.ResourceSelector.Property)),
						In:   []*string{to.Ptr(spec.ResourceSelector.Value)},
					},
				},
			},
		}
	}

	logger.Info("Creating or updating Azure Policy Exemption", "name", exemptionName, "scope", spec.Scope)

	resp, err := s.factory.Exemptions.CreateOrUpdate(ctx, spec.Scope, exemptionName, params, nil)
	if err != nil {
		return "", err
	}

	return *resp.Exemption.ID, nil
}

func (s *Service) Delete(ctx context.Context, scope string, exemptionID string) error {
	logger := log.FromContext(ctx)

	parts := strings.Split(exemptionID, "/")
	exemptionName := parts[len(parts)-1]

	logger.Info("Deleting Azure Policy Exemption", "name", exemptionName, "scope", scope)

	_, err := s.factory.Exemptions.Delete(ctx, scope, exemptionName, nil)
	return err
}
