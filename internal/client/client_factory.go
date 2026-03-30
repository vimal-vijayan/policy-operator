package client

import (
	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2"
	"github.com/vimal-vijayan/azure-policy-operator/internal/assignments"
	"github.com/vimal-vijayan/azure-policy-operator/internal/definitions"
	"github.com/vimal-vijayan/azure-policy-operator/internal/exemptions"
	"github.com/vimal-vijayan/azure-policy-operator/internal/initiatives"
)

type ARMClient struct {
	credential      azcore.TokenCredential
	SubscriptionID  string
	Definitions     definitions.API
	Initiatives     initiatives.API
	Assignments     assignments.API
	Exemptions      exemptions.API
	RoleAssignments *armauthorization.RoleAssignmentsClient
	RoleDefinitions *armauthorization.RoleDefinitionsClient
}
