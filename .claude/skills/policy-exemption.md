---
name: policy-exemption
description: This skill focuses on managing Azure Policy Exemptions, which allow for specific resources or scopes to be exempted from the effects of Azure Policy Assignments. It includes creating, updating, and deleting policy exemptions, as well as handling their parameters and metadata.
---

## Purpose
Used when creating or modifying the kubernetes operator logic or CR manifests for azure policy exemption. This includes defining the structure of the policy exemption, handling its parameters, and ensuring it is correctly represented in both the Kubernetes CRD and the Azure API.

## CRD Shape
```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyExemption
metadata:
  name: example-policy-exemption
spec:
  displayName: "Example Policy Exemption"
  policyAssignmentId: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Authorization/policyAssignments/{policyAssignmentName}"
  scope: "/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}"
  exemptionCategory: "Waiver"
  resourceSelector:
    property: "resourceType"
    operator: "Equals"
    value: "Microsoft.Compute/virtualMachines"
  description: "This is an example policy exemption."
  expiresOn: "2024-12-31T23:59:59Z"
```

### New kubebuilder api
- use the kubebuilder command create new api for the policy exemption with the name AzurePolicyExemption and group governance.platform.io and version v1alpha1
- define the spec and status fields in the api according to the CRD shape defined above

### Programming instructions
- Use the Azure SDK for Go to interact with the Azure Policy REST API for creating, updating, and deleting policy exemptions.
- azure sdk umbrella : https://pkg.go.dev/github.com/Azure/azure-sdk-for-go#section-readme
- azure sdk for arm policy: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy


### service - policy exemption
- path: internal/services/policyexemption/
- CreatePolicyExemption: Create a new Azure Policy Exemption, the business logic should be handled here, including validation of input parameters and interaction with the Azure API.
- UpdatePolicyExemption: Update an existing Azure Policy Exemption, ensuring that any changes are correctly reflected in both the Kubernetes CRD and the Azure API.
- DeletePolicyExemption: Remove an Azure Policy Exemption, ensuring that it is deleted from both the Kubernetes CRD and the Azure API.


### API
- path: internal/exemptions/
- Implement the API interface for managing the policy exemption
- Create function will create the policy exemption in azure and update the status of the CR accordingly
- use the google uuid package to generate a unique name for the policy exemption in azure
- Update function will update the policy exemption in azure and update the status of the CR accordingly
- Delete function will delete the policy exemption in azure and update the status of the CR accordingly


### Tests
- Implement unit tests for the controller logic, ensuring that the Create, Update, and Delete functions for the policy exemption work as expected. 
