---
name: policy-definition
description: This skill focuses on managing Azure Policy Definitions, which are the core building blocks of Azure governance. It includes creating, updating, and deleting policy definitions, as well as handling their parameters and metadata.
---

## Purpose
Used when creating or modifying the kubernetes operator logic or CR manifests for azure policy definition. This includes defining the structure of the policy definition, handling its parameters, and ensuring it is correctly represented in both the Kubernetes CRD and the Azure API.

## CRD Shape
```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: example-policy-definition
spec:
  displayName: "Example Policy Definition"
  description: "This is an example policy definition."
  version: "1.0.0"
  mode: Indexed
  metatadata:
    category: Governance
  parameters:
    allowedLocations:
      type: Array
      metadata:
        displayName: "Allowed Locations"
        description: "List of allowed locations for resources."
  policyRule:
    if:
        field: "[concat('tags[', parameters('tagName'), ']')]"
        exists: true
    then:
        effect: "deny"
  policyRuleJson: | --- OPTIONAL ---
    {
      "if": {
        "field": "[concat('tags[', parameters('tagName'), ']' )]",
        "exists": true
      },
      "then": {
        "effect": "deny"
      }
    } --- OPTIONAL ---
  managementGroupId: "/subscriptions/{subscriptionId}/providers/Microsoft.Management/managementGroups/{managementGroupName}" -- OPTIONAL ---
  subscriptionId: "00000000-0000-0000-0000-000000000000" -- OPTIONAL ---
```

### api definition requirement
- follow the CRD shape defined above for AzurePolicyDefinition, ensuring that all required fields are included and correctly typed. The CRD should allow users to define the policy rule using either a structured `policyRule` or a raw JSON string `policyRuleJson`, but not both. This design choice provides flexibility for users while maintaining clarity in how policies are defined and processed by the operator.
- policy definition can be defined with either a structured policyRule or a raw JSON policyRuleJson, but not both. This allows for flexibility in how users define their policies while ensuring that the operator can process them correctly.
- version should be included in the spec to allow for versioning of policy definitions, which is important for tracking changes and managing updates to policies over time. This field should follow semantic versioning (e.g., "1.0.0") to provide clarity on the nature of changes between versions.
- use the NewSetDefinitionVersionsClient from the Azure SDK for Go to manage different versions of policy definitions in Azure, allowing for better lifecycle management and version control of policies. This client provides methods for creating, updating, and retrieving specific versions of policy definitions, which can be crucial for maintaining backward compatibility and ensuring that changes to policies do not inadvertently affect existing resources.
- policyRuleJson should be optional and only used when the user prefers to define the policy rule as a raw JSON string. The operator should validate that if policyRuleJson is provided, then policyRule should not be defined, and vice versa. 
- policyRule should be defined as a JSON object that represents the logic of the policy. It should include the "if" and "then" conditions that specify when the policy should be applied and what effect it should have.
- parameters should be flexible to allow for different types of parameters (e.g., string, array, object) and should include metadata for display purposes in the Azure portal.
- preserve the user structure, do not over-model deeply unless required
- the definition should be on either management group or subscription level, not both. This means that the CRD should allow users to specify either a managementGroupId or a subscriptionId, but not both. This design choice ensures that the operator can correctly determine the scope of the policy definition and apply it accordingly in Azure.
- use apiextensionsv1.JSON or runtime.RawExtension for policyRule to allow for flexible and dynamic policy definitions without strict schema constraints. This allows users to define complex policies without being limited by a rigid CRD schema.
- in the status, include the policy definition ID from Azure, the provisioning state, and any error messages if the policy definition failed to create or update in Azure. This information is crucial for debugging and operational visibility.

### Programming instructions
- Use the Azure SDK for Go to interact with the Azure Policy REST API for creating, updating, and deleting policy definitions.
- azure sdk umbrella : https://pkg.go.dev/github.com/Azure/azure-sdk-for-go#section-readme
- azure sdk for arm policy: https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armpolicy