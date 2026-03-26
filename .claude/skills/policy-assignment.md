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
  notScopes:
    - "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}"
  parameters:
    allowedLocations:
      value: ["eastus", "westus"]
  metadata:
    category: Governance
  identity:
    type: "SystemAssigned"
    location: "westeurope"
    permissions:
      - role: "Contributor" -- OPTIONAL --
        roleDefinitionId: "/subscriptions/{subscriptionId}/providers/Microsoft.Authorization/roleDefinitions/{roleDefinitionId}" -- OPTIONAL --
        scope: "/subscriptions/{subscriptionId}"
  exemptions:
  - displayName: "Example Exemption"
    description: "This is an example exemption for the policy assignment."
    scope: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}"
    exemptionCategory: "Waiver"
    expiresOn: "2024-12-31T23:59:59Z"
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
- an assignment policy exemption should be added to the CRD as an optional attribute, allowing users to specify any exemptions that should be applied to the policy assignment. The operator should handle the logic for creating, updating, and deleting these exemptions in Azure as part of the overall management of the policy assignment. This feature provides users with greater flexibility in how they apply policies and manage exceptions within their Azure environment.
- the operator should add a exemptionId to the status of the CRD for each exemption created in Azure, allowing for better tracking and management of exemptions associated with the policy assignment. This information can be used to update or delete exemptions as needed when changes are made to the policy assignment or when the CRD is deleted.
- the operator is already managing the exemption CRD using the /internal/services/policyexemption/ service, reuse the functions in that service to manage the exemptions for the policy assignment, ensuring that the logic for handling exemptions is centralized and consistent across the operator.
- for the identity configuration, the permissions field is optional and should only be included if the user wants to specify custom permissions for the managed identity. If permissions are not provided ignore
- for the identity configuration, if the permissions field is included, the operator should validate the role if the role is provided (use the go sdk to get the role definition id), if the roleDefinitionId is provided, no validation is needed as the user is providing the full role definition id, either role or roleDefinitionId should be provided if permissions are included, if neither is provided, the operator should return an error indicating that at least one of these fields must be specified when permissions are included. This validation ensures that the operator has the necessary information to correctly configure the permissions for the managed identity associated with the policy assignment.
- api doc: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/authorization/armauthorization/v2#example-RoleAssignmentsClient.Create-CreateRoleAssignmentForResource

### API
- path: internal/assignments/
- Implement the API interface for assigning the policy
- Create function will create the policy assignment in azure and update the status of the CR accordingly
- use the google uuid package to generate a unique name for the policy assignment in azure
- Update function will update the policy assignment in azure and update the status of the CR accordingly
- Delete function will delete the policy assignment in azure and update the status of the CR accordingly