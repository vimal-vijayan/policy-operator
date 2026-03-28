---
title: "Policy Initiatives"
description: "Group related Azure Policy definitions into initiatives (policy sets) using AzurePolicyInitiative."
weight: 20
---

The `AzurePolicyInitiative` custom resource maps to an [Azure Policy Set Definition](https://learn.microsoft.com/en-us/azure/governance/policy/concepts/initiative-definition-structure). An initiative groups one or more policy definitions so they can be assigned and tracked together. The operator reconciles each resource by creating or updating the corresponding policy set definition in Azure.

{{< api-schema kind="AzurePolicyInitiative" version="v1alpha1" examples="4" status="true" >}}

{{< api-field name="apiVersion" type="String" desc="API version for this resource. Must be policy.azure.com/v1alpha1." >}}
```yaml
apiVersion: policy.azure.com/v1alpha1
```
{{< /api-field >}}

{{< api-field name="kind" type="String" desc="Resource kind. Must be AzurePolicyInitiative." >}}
```yaml
kind: AzurePolicyInitiative
```
{{< /api-field >}}

{{< api-field name="metadata" type="Object" children="true" desc="Standard Kubernetes object metadata." >}}
  {{< api-field name="name" type="String" required="true" desc="Unique name of the resource within its namespace." >}}
```yaml
metadata:
  name: security-baseline-initiative
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

{{< api-field name="spec" type="Object" required="true" children="true" desc="Desired state of the AzurePolicyInitiative." >}}

  {{< api-field name="displayName" type="String" required="true" desc="Human-readable name shown in the Azure portal and in compliance reports." >}}
```yaml
spec:
  displayName: "Security Baseline Initiative"
```
  {{< /api-field >}}

  {{< api-field name="description" type="String" desc="A human-readable explanation of what the initiative enforces and why." >}}
```yaml
spec:
  description: "Enforces a security baseline across all subscriptions."
```
  {{< /api-field >}}

  {{< api-field name="version" type="String" desc="Semantic version of the initiative (e.g. 1.0.0). Written into Azure Policy metadata under the version key; takes precedence over any version key in spec.metadata." >}}
```yaml
spec:
  version: "1.0.0"
```
  {{< /api-field >}}

  {{< api-field name="metadata" type="Object" desc="Arbitrary metadata attached to the Azure Policy set definition as a raw JSON object. Common keys: category, version." >}}
```yaml
spec:
  metadata:
    category: "Security"
    version: "1.0.0"
```
  {{< /api-field >}}

  {{< api-field name="parameters" type="Array" children="true" desc="Declares the parameters accepted by the initiative. Each entry defines a parameter that can be passed through to member policy definitions at assignment time." >}}
    {{< api-field name="name" type="String" required="true" desc="The parameter name referenced inside policyDefinitions[].parameters." >}}
```yaml
spec:
  parameters:
    - name: effect
```
    {{< /api-field >}}
    {{< api-field name="type" type="String" required="true" enum="String|Array|Object|Boolean|Integer|Float|DateTime" desc="Data type of the parameter." >}}
```yaml
spec:
  parameters:
    - name: effect
      type: String
```
    {{< /api-field >}}
    {{< api-field name="defaultValue" type="Object" desc="Default value used when no value is supplied at assignment time." >}}
```yaml
spec:
  parameters:
    - name: effect
      type: String
      defaultValue: "Audit"
```
    {{< /api-field >}}
    {{< api-field name="allowedValues" type="Array" desc="Restricts the set of values that can be supplied for this parameter." >}}
```yaml
spec:
  parameters:
    - name: effect
      type: String
      allowedValues: ["Audit", "Deny", "Disabled"]
```
    {{< /api-field >}}
    {{< api-field name="strongType" type="String" desc="Azure portal UI hint for this parameter (e.g. location, resourceTypes). Enables portal dropdowns and pickers." >}}
```yaml
spec:
  parameters:
    - name: allowedLocations
      type: Array
      strongType: "location"
```
    {{< /api-field >}}
  {{< /api-field >}}

  {{< api-field name="policyDefinitions" type="Array" required="true" children="true" desc="The list of policy definitions included in the initiative. At least one entry is required. Each entry must specify exactly one of policyDefinitionId or policyDefinitionRef." >}}
    {{< api-field name="policyDefinitionId" type="String" mutual="policyDefinitionRef" desc="The full Azure resource ID of a built-in or externally managed custom policy definition. Exactly one of policyDefinitionId or policyDefinitionRef must be specified." >}}
```yaml
spec:
  policyDefinitions:
    - policyDefinitionId: >-
        /providers/Microsoft.Authorization/policyDefinitions/0a914e76-4921-4c19-b460-a2d36003525a
```
    {{< /api-field >}}
    {{< api-field name="policyDefinitionRef" type="String" mutual="policyDefinitionId" desc="The name of an AzurePolicyDefinition CR in the same namespace. The operator resolves this reference to the Azure resource ID at reconcile time. Exactly one of policyDefinitionId or policyDefinitionRef must be specified." >}}
```yaml
spec:
  policyDefinitions:
    - policyDefinitionRef: require-https-storage
```
    {{< /api-field >}}
    {{< api-field name="parameters" type="Object" desc="Parameter values passed to the referenced policy definition. Keys are parameter names; values follow the Azure Policy parameter value schema." >}}
```yaml
spec:
  policyDefinitions:
    - policyDefinitionRef: require-https-storage
      parameters:
        effect:
          value: "[parameters('effect')]"
```
    {{< /api-field >}}
  {{< /api-field >}}

  {{< api-field name="subscriptionId" type="String" desc="Azure Subscription ID to create the initiative in. When omitted and managementGroupId is not set, the operator's own subscription is used." >}}
```yaml
spec:
  subscriptionId: "00000000-0000-0000-0000-000000000000"
```
  {{< /api-field >}}

  {{< api-field name="managementGroupId" type="String" desc="Management Group ID to create the initiative in. Use this to share the initiative across multiple subscriptions." >}}
```yaml
spec:
  managementGroupId: "my-management-group"
```
  {{< /api-field >}}

{{< /api-field >}}

{{< /api-schema >}}

{{< api-examples >}}

### Security baseline with CR references

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyInitiative
metadata:
  name: security-baseline-initiative
  namespace: platform-policies
spec:
  displayName: "Security Baseline Initiative"
  description: "Enforces HTTPS on storage accounts and audits VMs with public IPs."
  version: "1.0.0"
  metadata:
    category: "Security"
  parameters:
    - name: storageEffect
      type: String
      defaultValue: "Audit"
      allowedValues: ["Audit", "Deny", "Disabled"]
  policyDefinitions:
    - policyDefinitionRef: require-https-storage
      parameters:
        effect:
          value: "[parameters('storageEffect')]"
    - policyDefinitionRef: audit-public-ip-on-vms
  subscriptionId: "00000000-0000-0000-0000-000000000000"
```

### Built-in policy definitions at management group scope

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyInitiative
metadata:
  name: cis-benchmark-initiative
  namespace: platform-policies
spec:
  displayName: "CIS Azure Benchmark (Subset)"
  description: "Applies a subset of CIS Azure Benchmark built-in policies."
  version: "2.0.0"
  metadata:
    category: "Regulatory Compliance"
  policyDefinitions:
    - policyDefinitionId: >-
        /providers/Microsoft.Authorization/policyDefinitions/0a914e76-4921-4c19-b460-a2d36003525a
    - policyDefinitionId: >-
        /providers/Microsoft.Authorization/policyDefinitions/404c3081-a854-4457-ae30-26a93ef643f9
  managementGroupId: "corp-management-group"
```

### Mixed CR and built-in definitions with initiative parameters

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyInitiative
metadata:
  name: tagging-and-location-initiative
  namespace: platform-policies
spec:
  displayName: "Tagging and Location Governance"
  description: "Enforces required tags and restricts deployments to approved regions."
  version: "1.2.0"
  metadata:
    category: "Tags"
  parameters:
    - name: tagEffect
      type: String
      defaultValue: "Deny"
      allowedValues: ["Audit", "Deny"]
    - name: allowedLocations
      type: Array
      strongType: "location"
      defaultValue: ["australiaeast", "australiasoutheast"]
  policyDefinitions:
    - policyDefinitionRef: require-cost-center-tag
      parameters:
        effect:
          value: "[parameters('tagEffect')]"
    - policyDefinitionId: >-
        /providers/Microsoft.Authorization/policyDefinitions/e56962a6-4747-49cd-b67b-bf8b01975c4c
      parameters:
        listOfAllowedLocations:
          value: "[parameters('allowedLocations')]"
  managementGroupId: "my-management-group"
```

### Minimal initiative (no parameters)

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyInitiative
metadata:
  name: basic-audit-initiative
  namespace: platform-policies
spec:
  displayName: "Basic Audit Initiative"
  policyDefinitions:
    - policyDefinitionRef: audit-public-ip-on-vms
    - policyDefinitionRef: require-https-storage
  subscriptionId: "00000000-0000-0000-0000-000000000000"
```

{{< /api-examples >}}

{{< api-status >}}

{{< api-field name="initiativeId" type="String" desc="Full Azure resource ID of the created or updated policy set definition, populated after the first successful reconcile." >}}
```yaml
status:
  initiativeId: >-
    /subscriptions/00000000-0000-0000-0000-000000000000
    /providers/Microsoft.Authorization/policySetDefinitions/security-baseline-initiative
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
| `Ready` | `True` | `ReconcileSucceeded` | Initiative is in sync with Azure |
| `Ready` | `False` | `ReconcileFailed` | Last reconcile failed; see message |
| `Ready` | `False` | `AzureAPIError` | ARM API returned an error |

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      message: "Policy initiative successfully reconciled"
      lastTransitionTime: "2024-01-15T10:30:00Z"
```
{{< /api-field >}}

{{< /api-status >}}
