---
name: policy-assignment
description: This skill focuses on managing Azure Policy Assignments, which are used to apply Azure Policy Definitions to specific scopes. It includes creating, updating, and deleting policy assignments, as well as handling their parameters and metadata.
---

## Purpose
Used when creating or modifying the kubernetes operator logic or CR manifests for azure policy assignment. This includes defining the structure of the policy assignment, handling its parameters, and ensuring it is correctly represented in both the Kubernetes CRD and the Azure API.

## CRD Shape
```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: example-policy-assignment
spec:
  displayName: "Example Policy Assignment"
  policyDefinitionRef: example-policy-definition
  policyDefinitionId: "/subscriptions/{subscriptionId}/providers/Microsoft.Authorization/policyDefinitions/{policyDefinitionName}"
  scope: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}"
  parameters:
    allowedLocations:
      value: ["eastus", "westus"]
  metadata:
    category: Governance
```

### Programming instructions
- Use the Azure SDK for Go to interact with the Azure Policy REST API for creating, updating, and deleting policy definitions.
- azure sdk umbrella : https://pkg.go.dev/github.com/Azure/azure-sdk-for-go#section-readme
- azure sdk for arm policy: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy


### service - policy assignment
- path: internal/services/policyassignment/
- CreatePolicyAssignment: Create a new Azure Policy Assignment, the business logic should be handled here, including validation of input parameters and interaction with the Azure API.
- UpdatePolicyAssignment: Update an existing Azure Policy Assignment, ensuring that any changes are correctly reflected in both the Kubernetes CRD and the Azure API.
- DeletePolicyAssignment: Remove an Azure Policy Assignment, ensuring that it is deleted from both the Kubernetes CRD and the Azure API.


### API
- path: internal/assignments/
- Implement the API interface for assigning the policy
- Create function will create the policy assignment in azure and update the status of the CR accordingly
- use the google uuid package to generate a unique name for the policy assignment in azure
- Update function will update the policy assignment in azure and update the status of the CR accordingly
- Delete function will delete the policy assignment in azure and update the status of the CR accordingly