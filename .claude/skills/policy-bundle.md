---
name: policy-bundle
description: This skill focuses on managing Azure Policy Definitions, which are the core building blocks of Azure governance. It includes creating, updating, and deleting policy definitions, as well as handling their parameters and metadata.
---

## Purpose
Used when creating or modifying the kubernetes operator logic or CR manifests for azure policy definition. This includes defining the structure of the policy definition, handling its parameters, and ensuring it is correctly represented in both the Kubernetes CRD and the Azure API.

## CRD Shape
```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyInitiative
metadata:
  name: example-policy-initiative
spec:
  displayName: "Example Policy Initiative"
  description: "This is an example policy initiative."
  version: "1.0.0"
  metatadata:
    category: Governance
  parameters: -- OPTIONAL --
    - name: tagName
      type: String
      defaultValue: "environment"
      allowedValues: ["environment", "costcenter", "owner"]
      strongType: location -- OPTIONAL --
  policyDefinitions:
  - policyDefinitionId: "/subscriptions/{subscriptionId}/providers/Microsoft.Authorization/policyDefinitions/{policyDefinitionName}"
    parameters:
      tagName:
        value: "environment" 
  - policyDefinitionId: "/subscriptions/{subscriptionId}/providers/Microsoft.Authorization/policyDefinitions/{policyDefinitionName2}"
    parameters:
      tagName:
        value: "costcenter"
  - policyDefinitionRef:
      name: "example-policy-definition" -- reference to another CR in the same cluster ---
      parameters:
        tagName:
          value: "environment"
  managementGroupId: "/subscriptions/{subscriptionId}/providers/Microsoft.Management/managementGroups/{managementGroupName}" -- OPTIONAL ---
  subscriptionId: "00000000-0000-0000-0000-000000000000" -- OPTIONAL ---
```

### api definition requirement
- follow the CRD shape defined above for AzurePolicyInitiative, and ensure that the AzurePolicyDefinition CRD is structured to support the fields referenced in the initiative, such as policyDefinitionId and parameters. This means that the AzurePolicyDefinition CRD should include fields for defining the policy rule, parameters, and metadata that can be referenced by initiatives. Additionally, the operator logic should be implemented to handle the creation and management of both policy definitions and initiatives, ensuring that they are correctly linked and that any changes to definitions are reflected in the initiatives that reference them.
- the operator should also include the logic to reference the PolicyDefinition CRs when creating the Initiative, the PolicyDefinitionRef field, with the name should be used to lookup the corresponding PolicyDefinition CR in the cluster and extract the necessary information to include in the Initiative definition sent to Azure.
- 

### Programming instructions
- Use the Azure SDK for Go to interact with the Azure Policy REST API for creating, updating, and deleting policy definitions.
- azure sdk umbrella : https://pkg.go.dev/github.com/Azure/azure-sdk-for-go#section-readme
- azure sdk for arm policy: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy
- azure sdk function : func (client *SetDefinitionsClient) CreateOrUpdate(ctx context.Context, policySetDefinitionName string, parameters SetDefinition, options *SetDefinitionsClientCreateOrUpdateOptions) (SetDefinitionsClientCreateOrUpdateResponse, error)