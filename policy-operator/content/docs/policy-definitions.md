---
title: "Policy Definitions"
description: "Define and manage Azure Policy definitions as Kubernetes custom resources using AzurePolicyDefinition."
weight: 10
---

The `AzurePolicyDefinition` custom resource maps directly to an [Azure Policy Definition](https://learn.microsoft.com/en-us/azure/governance/policy/overview). The operator reconciles each resource by creating or updating the corresponding definition in Azure.

{{< api-schema kind="AzurePolicyDefinition" version="v1alpha1" examples="4" status="true" >}}

{{< api-field name="apiVersion" type="String" desc="API version for this resource. Must be policy.azure.com/v1alpha1." >}}
```yaml
apiVersion: policy.azure.com/v1alpha1
```
{{< /api-field >}}

{{< api-field name="kind" type="String" desc="Resource kind. Must be AzurePolicyDefinition." >}}
```yaml
kind: AzurePolicyDefinition
```
{{< /api-field >}}

{{< api-field name="metadata" type="Object" children="true" desc="Standard Kubernetes object metadata." >}}
  {{< api-field name="name" type="String" required="true" desc="Unique name of the resource within its namespace." >}}
```yaml
metadata:
  name: require-https-storage
```
  {{< /api-field >}}
  {{< api-field name="namespace" type="String" desc="Kubernetes namespace." >}}
```yaml
metadata:
  namespace: platform-policies
```
  {{< /api-field >}}
  {{< api-field name="labels" type="Object" desc="Map of key/value pairs for organizing resources." >}}
```yaml
metadata:
  labels:
    team: platform
    env: production
```
  {{< /api-field >}}
  {{< api-field name="annotations" type="Object" desc="Arbitrary non-identifying metadata." >}}
```yaml
metadata:
  annotations:
    policy.azure.com/owner: platform-team
```
  {{< /api-field >}}
{{< /api-field >}}

{{< api-field name="spec" type="Object" required="true" children="true" desc="Desired state of the AzurePolicyDefinition." >}}

  {{< api-field name="displayName" type="String" required="true" desc="Human-readable name shown in the Azure portal and in compliance reports." >}}
```yaml
spec:
  displayName: "Require HTTPS on Storage Accounts"
```
  {{< /api-field >}}

  {{< api-field name="description" type="String" desc="A human-readable explanation of what the policy enforces and why." >}}
```yaml
spec:
  description: "Audits storage accounts that do not enforce HTTPS-only traffic."
```
  {{< /api-field >}}

  {{< api-field name="policyType" type="String" default="Custom" enum="BuiltIn|Custom|NotSpecified|Static" desc="Classifies the policy definition." >}}
```yaml
spec:
  policyType: Custom
```
  {{< /api-field >}}

  {{< api-field name="mode" type="String" required="true" enum="All|Indexed" desc="Controls which resource types the policy evaluates. All includes resource groups and subscriptions; Indexed targets only types that support tags and location." >}}
```yaml
spec:
  mode: Indexed   # tag-aware resources only
  # mode: All     # includes resource groups and subscriptions
```
  {{< /api-field >}}

  {{< api-field name="version" type="String" desc="Semantic version of the policy definition (e.g. 1.0.0). Written into Azure Policy metadata under the version key; takes precedence over any version key in spec.metadata." >}}
```yaml
spec:
  version: "1.0.0"
```
  {{< /api-field >}}

  {{< api-field name="metadata" type="Object" desc="Arbitrary metadata attached to the Azure Policy definition as a raw JSON object. Common keys: category, version." >}}
```yaml
spec:
  metadata:
    category: "Storage"
    version: "1.0.0"
```
  {{< /api-field >}}

  {{< api-field name="parameters" type="Object" desc="Declares the parameters that the policy rule can reference. Each key is a parameter name following the Azure Policy parameter definition schema." >}}
```yaml
parameters:
  effect:
    type: String
    metadata:
      displayName: Effect
    allowedValues: [Audit, Deny, Disabled]
    defaultValue: Audit
```
  {{< /api-field >}}

  {{< api-field name="policyRule" type="Object" required="true" mutual="policyRuleJson" children="true" desc="The policy logic as an inline object. Exactly one of policyRule or policyRuleJson must be specified." >}}
    {{< api-field name="if" type="Object" required="true" desc="Condition that evaluates the resource. Supports field, allOf, anyOf, not, and count expressions." >}}
```yaml
spec:
  policyRule:
    if:
      allOf:
        - field: "type"
          equals: "Microsoft.Storage/storageAccounts"
        - field: "Microsoft.Storage/storageAccounts/supportsHttpsTrafficOnly"
          notEquals: true
```
    {{< /api-field >}}
    {{< api-field name="then" type="Object" required="true" desc="Effect to apply when the condition is met (e.g. Audit, Deny, Modify, DeployIfNotExists)." >}}
```yaml
spec:
  policyRule:
    then:
      effect: "[parameters('effect')]"
```
    {{< /api-field >}}
  {{< /api-field >}}

  {{< api-field name="policyRuleJson" type="String" required="true" mutual="policyRule" desc="The policy rule as a raw JSON string. Use this when importing large pre-built definitions where inline YAML is impractical. Exactly one of policyRule or policyRuleJson must be specified." >}}
```yaml
spec:
  policyRuleJson: |
    {
      "if": {
        "allOf": [
          { "field": "type", "equals": "Microsoft.Compute/virtualMachines" },
          { "field": "location", "notIn": ["australiaeast", "australiasoutheast"] }
        ]
      },
      "then": { "effect": "Deny" }
    }
```
  {{< /api-field >}}

  {{< api-field name="subscriptionId" type="String" desc="Azure Subscription ID to create the policy definition in. When omitted, the definition is created at management group scope (requires managementGroupId)." >}}
```yaml
spec:
  subscriptionId: "00000000-0000-0000-0000-000000000000"
```
  {{< /api-field >}}

  {{< api-field name="managementGroupId" type="String" desc="Management Group ID to create the policy definition in. Use this to share definitions across multiple subscriptions." >}}
```yaml
spec:
  managementGroupId: "my-management-group"
```
  {{< /api-field >}}

{{< /api-field >}}

{{< /api-schema >}}

{{< api-examples >}}

### Audit storage accounts without HTTPS

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: require-https-storage
spec:
  displayName: "Require HTTPS on Storage Accounts"
  description: "Audits storage accounts that do not enforce HTTPS-only traffic."
  policyType: Custom
  mode: Indexed
  version: "1.0.0"
  metadata:
    category: "Storage"
  parameters:
    effect:
      type: String
      metadata:
        displayName: Effect
      allowedValues: [Audit, Deny, Disabled]
      defaultValue: Audit
  policyRule:
    if:
      allOf:
        - field: "type"
          equals: "Microsoft.Storage/storageAccounts"
        - field: "Microsoft.Storage/storageAccounts/supportsHttpsTrafficOnly"
          notEquals: true
    then:
      effect: "[parameters('effect')]"
  subscriptionId: "00000000-0000-0000-0000-000000000000"
```

### Deny untagged resources at management group scope

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: require-cost-center-tag
spec:
  displayName: "Require CostCenter tag on resources"
  description: "Denies creation of resources missing the CostCenter tag."
  policyType: Custom
  mode: Indexed
  version: "2.0.1"
  metadata:
    category: "Tags"
  policyRule:
    if:
      field: "tags['CostCenter']"
      exists: false
    then:
      effect: "Deny"
  managementGroupId: "my-management-group"
```

### Import a large policy from raw JSON

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: restrict-vm-regions
spec:
  displayName: "Restrict VM deployments to approved regions"
  mode: All
  policyRuleJson: |
    {
      "if": {
        "allOf": [
          { "field": "type", "equals": "Microsoft.Compute/virtualMachines" },
          { "field": "location", "notIn": ["australiaeast", "australiasoutheast"] }
        ]
      },
      "then": { "effect": "Deny" }
    }
  subscriptionId: "00000000-0000-0000-0000-000000000000"
```

### Audit VMs with public IPs (no parameters)

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: audit-public-ip-on-vms
spec:
  displayName: "Audit VMs with public IP addresses"
  mode: Indexed
  policyRule:
    if:
      allOf:
        - field: "type"
          equals: "Microsoft.Network/networkInterfaces"
        - field: "Microsoft.Network/networkInterfaces/ipconfigurations[*].publicIpAddress.id"
          exists: true
    then:
      effect: "Audit"
  managementGroupId: "platform-management-group"
```

{{< /api-examples >}}

{{< api-status >}}

{{< api-field name="policyDefinitionId" type="String" desc="Full Azure resource ID of the created or updated policy definition, populated after the first successful reconcile." >}}
```yaml
status:
  policyDefinitionId: >-
    /subscriptions/00000000-0000-0000-0000-000000000000
    /providers/Microsoft.Authorization/policyDefinitions/require-https-storage
```
{{< /api-field >}}

{{< api-field name="appliedVersion" type="String" desc="The spec.version value last successfully written to Azure Policy metadata. Mirrors spec.version after each successful reconcile." >}}
```yaml
status:
  appliedVersion: "1.0.0"
```
{{< /api-field >}}

{{< api-field name="conditions" type="Array" desc="Standard Kubernetes conditions reflecting the current reconcile state." >}}

| Type | Status | Reason | Meaning |
|---|---|---|---|
| `Ready` | `True` | `ReconcileSucceeded` | Definition is in sync with Azure |
| `Ready` | `False` | `ReconcileFailed` | Last reconcile failed; see message |
| `Ready` | `False` | `AzureAPIError` | ARM API returned an error |

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      message: "Policy definition successfully reconciled"
      lastTransitionTime: "2024-01-15T10:30:00Z"
```
{{< /api-field >}}

{{< /api-status >}}
