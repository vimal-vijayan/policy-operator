---
title: "Policy Definitions"
description: "Define and manage Azure Policy definitions as Kubernetes custom resources using AzurePolicyDefinition."
weight: 10
---

The `AzurePolicyDefinition` custom resource maps directly to an [Azure Policy Definition](https://learn.microsoft.com/en-us/azure/governance/policy/overview). The operator reconciles each resource by creating or updating the corresponding definition in Azure.

{{< api-schema kind="AzurePolicyDefinition" version="v1alpha1" examples="6" status="true" tips="true" >}}

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
  {{< api-field name="annotations" type="Object" children="true" desc="Arbitrary non-identifying metadata. The following annotations control import behaviour when adopting an existing Azure Policy Definition." >}}
    {{< api-field name="governance.platform.io/import-id" type="String" desc="Full Azure resource ID of the existing policy definition to adopt. Required when importing. Supports both subscription and management group scoped IDs." >}}
```yaml
metadata:
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/my-policy
```
    {{< /api-field >}}
    {{< api-field name="governance.platform.io/import-name" type="String" desc="The bare Azure policy definition name (last segment of the resource ID). Used as the policy name when writing back to Azure in reconcile or once mode." >}}
```yaml
metadata:
  annotations:
    governance.platform.io/import-name: "my-policy"
```
    {{< /api-field >}}
    {{< api-field name="governance.platform.io/import-mode" type="String" default="observe-only" enum="observe-only|reconcile|adopt-once" desc="Controls what happens after the definition is imported. observe-only: read-only, no changes are pushed to Azure. reconcile: continuously sync Azure to match the spec on every reconcile. adopt-once: adopt and apply changes once, then stop reconciling." >}}
```yaml
metadata:
  annotations:
    governance.platform.io/import-mode: "observe-only"  # read-only
    # governance.platform.io/import-mode: "reconcile"   # continuously sync
    # governance.platform.io/import-mode: "adopt-once"  # adopt once
```
    {{< /api-field >}}
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

### Import existing definition (observe-only)

Adopt an existing Azure Policy Definition without making any changes to Azure. The operator fetches the live definition and reports any drift between the Azure state and the spec.

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: require-tag-on-resources
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/require-tag-on-resources
    governance.platform.io/import-name: "require-tag-on-resources"
    governance.platform.io/import-mode: "observe-only"
spec:
  displayName: "Require a tag on resources"
  description: "Enforces the existence of a required tag on all resources."
  policyType: Custom
  mode: Indexed
  metadata:
    category: "Tags"
  version: "1.0.0"
  parameters:
    tagName:
      type: String
      metadata:
        displayName: "Tag Name"
        description: "Name of the tag that must exist on the resource."
  policyRule:
    if:
      field: "[concat('tags[', parameters('tagName'), ']')]"
      exists: "false"
    then:
      effect: "deny"
  subscriptionId: "00000000-0000-0000-0000-000000000000"
```

### Import and reconcile existing definition (policyRuleJson)

Adopt an existing definition and continuously sync Azure to match the spec. Uses `policyRuleJson` for definitions where the rule is more conveniently expressed as raw JSON.

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: allowed-locations-policy
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/allowed-locations-policy
    governance.platform.io/import-name: "allowed-locations-policy"
    governance.platform.io/import-mode: "reconcile"
spec:
  displayName: "Allowed Locations"
  description: "Enforces that resources are only deployed in allowed locations."
  policyType: Custom
  mode: Indexed
  metadata:
    category: "General"
  version: "1.0.0"
  parameters:
    allowedLocations:
      type: Array
      defaultValue:
        - "eastus2"
      metadata:
        displayName: "Allowed locations"
        description: "The list of allowed locations for resources."
        strongType: "location"
  policyRuleJson: |
    {
      "if": {
        "not": {
          "field": "location",
          "in": "[parameters('allowedLocations')]"
        }
      },
      "then": {
        "effect": "audit"
      }
    }
  subscriptionId: "00000000-0000-0000-0000-000000000000"
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
| `Ready` | `True` | `Reconciled` | Definition is in sync with Azure |
| `Ready` | `True` | `ObservedOnly` | Imported in observe-only mode; no changes pushed to Azure |
| `Ready` | `False` | `ReconcileFailed` | Last reconcile failed; see message |
| `Ready` | `False` | `DeleteFailed` | Finalizer cleanup failed; see message |
| `Ready` | `False` | `ImportFailed` | Import from Azure failed; see message |
| `Ready` | `False` | `ImportConflict` | `import-id` annotation conflicts with already-bound `policyDefinitionId` |
| `Imported` | `True` | `ImportSucceeded` | Existing Azure definition was adopted successfully |
| `Imported` | `False` | `ImportFailed` | Could not fetch or validate the definition from Azure |
| `DriftDetected` | `True` | `SpecMismatch` | Live Azure definition differs from the desired spec; see message for fields |
| `DriftDetected` | `False` | `InSync` | Azure definition matches the desired spec |

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: Reconciled
      message: "Policy definition successfully reconciled"
      lastTransitionTime: "2024-01-15T10:30:00Z"
    - type: Imported
      status: "True"
      reason: ImportSucceeded
      message: "Existing Azure Policy Definition was adopted successfully."
      lastTransitionTime: "2024-01-15T10:30:00Z"
    - type: DriftDetected
      status: "True"
      reason: SpecMismatch
      message: "Live Azure definition differs from desired spec: displayName, description"
      lastTransitionTime: "2024-01-15T10:30:00Z"
```
{{< /api-field >}}

{{< /api-status >}}

{{< api-tips >}}

<div class="tips-list">

  <div class="tips-item">
    <button class="tips-item__header" data-api-toggle aria-expanded="false" aria-controls="tip-import-definitions" type="button">
      Importing Azure Policy Definitions
      <svg class="tips-item__chevron" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="4 6 8 10 12 6"/></svg>
    </button>
    <div class="tips-item__body" id="tip-import-definitions" hidden>
      <p>Look, nobody is going to sit there and hand-craft YAML for hundreds of custom policies that already exist in Azure. That's what AI is for. Seriously — you've got Claude skills, GitHub Copilot rules, and I'm pretty sure your IDE already has an AI assistant quietly judging your indentation. Put them to work.</p>
      <p>Export your existing policy definition, drop it in front of your AI assistant, and ask it to generate the <code>AzurePolicyDefinition</code> manifest. It will fill in <code>policyRule</code>, <code>parameters</code>, <code>displayName</code>, and everything else while you grab a coffee.</p>
<div class="highlight"><pre><code class="language-bash"># Export from Azure CLI, hand it to your AI assistant
az policy definition show --name "my-custom-policy" --subscription "00000000-0000-0000-0000-000000000000"</code></pre></div>
      <h3>Always start with <code>observe-only</code></h3>
      <p>Before you let the operator touch anything, import with <code>observe-only</code> mode. It reads the live definition from Azure and reports drift — no changes pushed.</p>
<div class="highlight"><pre><code class="language-yaml">metadata:
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/my-policy
    governance.platform.io/import-name: "my-policy"
    governance.platform.io/import-mode: "observe-only"</code></pre></div>
      <h3>Watch status and drift with kubectl</h3>
      <p>Once deployed, use kubectl to see what the operator sees:</p>
<div class="highlight"><pre><code class="language-bash">kubectl describe azurepolicydefinition my-policy
kubectl get azurepolicydefinition my-policy -o yaml</code></pre></div>
      <p>Watch the <code>DriftDetected</code> condition — if <code>reason: SpecMismatch</code> appears, your spec and the live Azure definition differ. You'll know exactly which fields before committing to anything.</p>
      <h3>When you're confident, choose your next move</h3>
      <p>Switch to <strong><code>reconcile</code></strong> for continuous GitOps management — the operator keeps Azure in sync with your spec on every reconcile. Or switch to <strong><code>adopt-once</code></strong> to adopt and apply your spec a single time, then step back.</p>
<div class="highlight"><pre><code class="language-yaml">governance.platform.io/import-mode: "reconcile"   # continuous sync
governance.platform.io/import-mode: "adopt-once"  # adopt once, then hands off</code></pre></div>
      <div class="callout callout-warning" style="margin-top:1.25rem">
        <div class="callout-title">Never remove import annotations</div>
        <p>The operator maps the CRD <code>metadata.name</code> to the Azure policy name by default. Import is the exception — <code>governance.platform.io/import-name</code> is used directly instead. Removing these annotations causes the operator to fall back to the CRD name, losing track of the original definition and potentially creating a duplicate. Removing import annotations from a managed policy can result in an unpleasant policy management experience.</p>
      </div>
    </div>
  </div>

</div>

{{< /api-tips >}}
