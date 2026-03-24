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
- policyDefinitionRef in the CRD should be used to reference the AzurePolicyDefinition CRD, allowing for a clear link between the policy assignment and its corresponding policy definition. The operator should resolve this reference to retrieve the necessary information about the policy definition and use the policyDefinitionId from the status of the AzurePolicyDefinition CRD when creating or updating the policy assignment in Azure. This design choice promotes a clear separation of concerns and allows for better management of policy definitions and their assignments within the Kubernetes environment.
- either policyDefinitionRef or policyDefinitionId should be required in the CRD, but not both. This allows for flexibility in how users define their policy assignments while ensuring that the operator can correctly resolve the policy definition information when interacting with the Azure API. If policyDefinitionRef is provided, the operator should resolve it to get the policyDefinitionId from the status of the referenced AzurePolicyDefinition CRD. If policyDefinitionId is provided directly, the operator can use it without needing to resolve a reference. This design choice simplifies the user experience while maintaining the necessary functionality for managing policy assignments effectively.


### API
- path: internal/assignments/
- Implement the API interface for assigning the policy
- Create function will create the policy assignment in azure and update the status of the CR accordingly
- use the google uuid package to generate a unique name for the policy assignment in azure
- Update function will update the policy assignment in azure and update the status of the CR accordingly
- Delete function will delete the policy assignment in azure and update the status of the CR accordingly