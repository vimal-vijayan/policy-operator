// Package client provides Azure SDK interactions for managing Azure Policy resources.
package client

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy"
	"github.com/vimal-vijayan/azure-policy-operator/internal/assignments"
	"github.com/vimal-vijayan/azure-policy-operator/internal/exemptions"
)

func New(subscriptionID string) (*ARMClient, error) {
	cred, err := azidentity.NewDefaultAzureCredential(nil)
	if err != nil {
		return nil, err
	}

	definitionsClient, err := armpolicy.NewDefinitionsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	initiativesClient, err := armpolicy.NewSetDefinitionsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	assignmentsClient, err := armpolicy.NewAssignmentsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	exemptionsClient, err := armpolicy.NewExemptionsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	roleAssignmentsClient, err := armauthorization.NewRoleAssignmentsClient(subscriptionID, cred, nil)
	if err != nil {
		return nil, err
	}

	roleDefinitionsClient, err := armauthorization.NewRoleDefinitionsClient(cred, nil)
	if err != nil {
		return nil, err
	}

	return &ARMClient{
		credential:      cred,
		SubscriptionID:  subscriptionID,
		Definitions:     definitionsClient,
		Initiatives:     initiativesClient,
		Assignments:     assignments.NewClient(assignmentsClient),
		Exemptions:      exemptions.NewClient(exemptionsClient),
		RoleAssignments: roleAssignmentsClient,
		RoleDefinitions: roleDefinitionsClient,
	}, nil
}
