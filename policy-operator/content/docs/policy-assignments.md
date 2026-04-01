---
title: "Policy Assignments"
description: "Assign Azure Policy definitions and initiatives to subscriptions, resource groups, or management groups using AzurePolicyAssignment."
weight: 30
---

The `AzurePolicyAssignment` custom resource maps to an [Azure Policy Assignment](https://learn.microsoft.com/en-us/azure/governance/policy/concepts/assignment-structure). An assignment binds a policy definition or initiative to a scope (subscription, resource group, or management group) and optionally configures parameters, managed identity, resource selectors, and inline exemptions. The operator reconciles each resource by creating or updating the corresponding assignment in Azure.

{{< api-schema kind="AzurePolicyAssignment" version="v1alpha1" examples="6" status="true" tips="true" >}}

{{< api-field name="apiVersion" type="String" desc="API version for this resource. Must be policy.azure.com/v1alpha1." >}}
```yaml
apiVersion: policy.azure.com/v1alpha1
```
{{< /api-field >}}

{{< api-field name="kind" type="String" desc="Resource kind. Must be AzurePolicyAssignment." >}}
```yaml
kind: AzurePolicyAssignment
```
{{< /api-field >}}

{{< api-field name="metadata" type="Object" children="true" desc="Standard Kubernetes object metadata." >}}
  {{< api-field name="name" type="String" required="true" desc="Unique name of the resource within its namespace." >}}
```yaml
metadata:
  name: assign-security-baseline
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
  {{< api-field name="annotations" type="Object" children="true" desc="Arbitrary non-identifying metadata. The operator recognises specific annotations for import and observe-only mode." >}}
    {{< api-field name="governance.platform.io/import-id" type="String" desc="Azure resource ID of an existing policy assignment to import into operator management. When set, the operator adopts the existing Azure assignment instead of creating a new one." >}}
```yaml
metadata:
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyAssignments/my-existing-assignment
```
    {{< /api-field >}}
    {{< api-field name="governance.platform.io/import-mode" type="String" enum="observe-only|adopt-once|reconcile" default="observe-only" desc="Controls what the operator does after adopting the existing Azure assignment. Must be used alongside governance.platform.io/import-id. If omitted, defaults to observe-only. See the import modes table below." >}}

| Mode | Behaviour |
|---|---|
| `observe-only` | Reads the assignment from Azure and populates status. The operator never creates, updates, or deletes the Azure resource. Useful for auditing or brownfield visibility without taking ownership. |
| `adopt-once` | Adopts the existing assignment on the first reconcile, then immediately begins managing it — subsequent reconciles call CreateOrUpdate to converge spec with Azure. |
| `reconcile` | Same as `adopt-once`. Adopts the assignment and continuously reconciles spec changes against Azure going forward. |

```yaml
metadata:
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyAssignments/my-existing-assignment
    governance.platform.io/import-mode: "observe-only"  # or: adopt-once, reconcile
```
    {{< /api-field >}}
    {{< api-field name="governance.platform.io/import-name" type="String" desc="The bare Azure policy assignment name (last segment of the resource ID). Used as the assignment name when writing back to Azure in reconcile or adopt-once mode. Must be set when the Kubernetes resource name differs from the Azure assignment name." >}}
```yaml
metadata:
  annotations:
    governance.platform.io/import-name: "785fcbc8d4df43f6a63ac030"
```
    {{< /api-field >}}
    {{< api-field name="(any)" type="String" desc="Any additional annotation key/value pair for organisational metadata." >}}
```yaml
metadata:
  annotations:
    policy.azure.com/owner: platform-team
```
    {{< /api-field >}}
  {{< /api-field >}}
{{< /api-field >}}

{{< api-field name="spec" type="Object" required="true" children="true" desc="Desired state of the AzurePolicyAssignment." >}}

  {{< api-field name="displayName" type="String" required="true" desc="Human-readable name shown in the Azure portal and in compliance reports." >}}
```yaml
spec:
  displayName: "Assign Security Baseline"
```
  {{< /api-field >}}

  {{< api-field name="description" type="String" desc="A human-readable explanation of why this assignment exists." >}}
```yaml
spec:
  description: "Enforces the security baseline across the production subscription."
```
  {{< /api-field >}}

  {{< api-field name="policyDefinitionRef" type="String" required="true" mutual="policyDefinitionId" desc="Name of an AzurePolicyDefinition or AzurePolicyInitiative CR in the same namespace. The operator resolves this reference to the Azure resource ID from the CR's status at reconcile time. Exactly one of policyDefinitionRef or policyDefinitionId must be specified." >}}
```yaml
spec:
  policyDefinitionRef: require-https-storage
```
  {{< /api-field >}}

  {{< api-field name="policyDefinitionId" type="String" required="true" mutual="policyDefinitionRef" desc="Full Azure resource ID of a built-in or externally managed policy definition or initiative. Exactly one of policyDefinitionRef or policyDefinitionId must be specified." >}}
```yaml
spec:
  policyDefinitionId: >-
    /providers/Microsoft.Authorization/policySetDefinitions/1f3afdf9-d0c9-4c3d-847f-89da613e70a8
```
  {{< /api-field >}}

  {{< api-field name="scope" type="String" required="true" desc="The Azure resource scope at which this assignment applies. Accepts subscription, resource group, or management group scope paths." >}}
```yaml
spec:
  # Subscription scope
  scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
  # Resource group scope
  # scope: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/my-rg"
  # Management group scope
  # scope: "/providers/Microsoft.Management/managementGroups/my-mg"
```
  {{< /api-field >}}

  {{< api-field name="notScopes" type="Array" desc="List of resource scopes excluded from evaluation. Resources at these scopes and their children are not evaluated against this assignment." >}}
```yaml
spec:
  notScopes:
    - "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/exempt-rg"
    - "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/legacy-rg"
```
  {{< /api-field >}}

  {{< api-field name="enforcementMode" type="String" default="Default" enum="Default|DoNotEnforce" desc="Controls whether the policy effect is applied. Default enforces the policy. DoNotEnforce evaluates compliance but does not block or modify resources — useful for testing or auditing before full enforcement." >}}
```yaml
spec:
  enforcementMode: Default
  # enforcementMode: DoNotEnforce  # evaluate but do not enforce
```
  {{< /api-field >}}

  {{< api-field name="parameters" type="Object" desc="Parameter values for the assigned policy definition or initiative. Keys are parameter names; values follow the Azure Policy parameter value schema ({\"value\": ...})." >}}
```yaml
spec:
  parameters:
    effect:
      value: "Deny"
    allowedLocations:
      value: ["australiaeast", "australiasoutheast"]
```
  {{< /api-field >}}

  {{< api-field name="metadata" type="Object" desc="Arbitrary metadata attached to the Azure Policy assignment as a raw JSON object." >}}
```yaml
spec:
  metadata:
    assignedBy: "platform-team"
    ticketRef: "INFRA-1234"
```
  {{< /api-field >}}

  {{< api-field name="identity" type="Object" children="true" desc="Managed identity configuration for the assignment. Required when the assigned policy uses deployIfNotExists or modify effects, as Azure needs an identity with appropriate permissions to remediate resources." >}}
    {{< api-field name="type" type="String" required="true" default="None" enum="SystemAssigned|UserAssigned|None" desc="The identity type. SystemAssigned creates an identity managed by Azure. UserAssigned uses an existing user-assigned managed identity. None attaches no identity." >}}
```yaml
spec:
  identity:
    type: SystemAssigned
```
    {{< /api-field >}}
    {{< api-field name="userAssignedIdentityId" type="String" desc="Full Azure resource ID of the user-assigned managed identity. Required when type is UserAssigned." >}}
```yaml
spec:
  identity:
    type: UserAssigned
    userAssignedIdentityId: >-
      /subscriptions/00000000-0000-0000-0000-000000000000
      /resourceGroups/my-rg/providers/Microsoft.ManagedIdentity
      /userAssignedIdentities/my-identity
```
    {{< /api-field >}}
    {{< api-field name="location" type="String" default="westeurope" desc="Azure region where the managed identity is created. Required for SystemAssigned or UserAssigned types. Must match a region where Azure Policy identity creation is supported." >}}
```yaml
spec:
  identity:
    type: SystemAssigned
    location: australiaeast
```
    {{< /api-field >}}
    {{< api-field name="permissions" type="Array" children="true" desc="Role assignments to create for the managed identity after the assignment is created. Each entry grants the identity a role at a given scope." >}}
      {{< api-field name="role" type="String" mutual="roleDefinitionId" desc="Built-in Azure role name (e.g. Contributor, Reader). The operator resolves this to a role definition ID via the Azure API. Either role or roleDefinitionId must be specified." >}}
```yaml
spec:
  identity:
    permissions:
      - role: Contributor
        scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
```
      {{< /api-field >}}
      {{< api-field name="roleDefinitionId" type="String" mutual="role" desc="Full Azure resource ID of the role definition. Either role or roleDefinitionId must be specified." >}}
```yaml
spec:
  identity:
    permissions:
      - roleDefinitionId: >-
          /subscriptions/00000000-0000-0000-0000-000000000000
          /providers/Microsoft.Authorization/roleDefinitions/b24988ac-6180-42a0-ab88-20f7382dd24c
        scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
```
      {{< /api-field >}}
      {{< api-field name="scope" type="String" required="true" desc="Azure resource scope at which the role assignment is created." >}}
```yaml
spec:
  identity:
    permissions:
      - role: Contributor
        scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
```
      {{< /api-field >}}
    {{< /api-field >}}
  {{< /api-field >}}

  {{< api-field name="nonComplianceMessages" type="Object" children="true" desc="Custom messages shown in the Azure portal when resources are flagged as non-compliant under this assignment." >}}
    {{< api-field name="default" type="String" desc="Fallback message used when no per-policy message matches. Shown for all non-compliant resources unless overridden." >}}
```yaml
spec:
  nonComplianceMessages:
    default: "This resource does not comply with the platform security baseline."
```
    {{< /api-field >}}
    {{< api-field name="perPolicy" type="Array" children="true" desc="Per-policy messages for individual policy references within an initiative. Has no effect when assigning a single policy definition." >}}
      {{< api-field name="policyReferenceId" type="String" required="true" desc="Policy definition reference ID from the assigned initiative." >}}
```yaml
spec:
  nonComplianceMessages:
    perPolicy:
      - policyReferenceId: RequireHttpsOnStorage
        message: "Storage accounts must enforce HTTPS-only traffic."
```
      {{< /api-field >}}
      {{< api-field name="message" type="String" required="true" desc="Non-compliance message for the referenced policy." >}}
```yaml
spec:
  nonComplianceMessages:
    perPolicy:
      - policyReferenceId: RequireHttpsOnStorage
        message: "Storage accounts must enforce HTTPS-only traffic."
```
      {{< /api-field >}}
    {{< /api-field >}}
  {{< /api-field >}}

  {{< api-field name="resourceSelectors" type="Array" children="true" desc="Filters the set of resources evaluated by the assignment based on resource properties. Useful for phased rollouts or targeting specific resource types or locations." >}}
    {{< api-field name="name" type="String" required="true" desc="Name of the resource selector." >}}
```yaml
spec:
  resourceSelectors:
    - name: storage-only
```
    {{< /api-field >}}
    {{< api-field name="selectors" type="Array" children="true" desc="List of selector expressions combined with AND logic." >}}
      {{< api-field name="property" type="String" required="true" enum="resourceType|resourceLocation|resourceWithoutLocation|userPrincipalId|groupPrincipalId" desc="The resource property to filter on." >}}
```yaml
spec:
  resourceSelectors:
    - name: storage-only
      selectors:
        - property: resourceType
          operator: In
          values: ["Microsoft.Storage/storageAccounts"]
```
      {{< /api-field >}}
      {{< api-field name="operator" type="String" required="true" enum="In|notIn" desc="Filter operator. In requires the resource property to match one of the values. notIn excludes matching resources." >}}
```yaml
spec:
  resourceSelectors:
    - name: approved-regions
      selectors:
        - property: resourceLocation
          operator: notIn
          values: ["global", ""]
```
      {{< /api-field >}}
      {{< api-field name="values" type="Array" required="true" desc="List of values to match against the selected property." >}}
```yaml
spec:
  resourceSelectors:
    - name: approved-regions
      selectors:
        - property: resourceLocation
          operator: In
          values: ["australiaeast", "australiasoutheast"]
```
      {{< /api-field >}}
    {{< /api-field >}}
  {{< /api-field >}}

  {{< api-field name="exemptions" type="Array" children="true" desc="Inline exemptions to create alongside this assignment. Each entry creates a corresponding AzurePolicyExemption in Azure scoped to this assignment." >}}
    {{< api-field name="displayName" type="String" required="true" desc="Human-readable name for the exemption." >}}
```yaml
spec:
  exemptions:
    - displayName: "Exempt legacy storage account"
```
    {{< /api-field >}}
    {{< api-field name="description" type="String" desc="A human-readable explanation for why this exemption exists." >}}
```yaml
spec:
  exemptions:
    - displayName: "Exempt legacy storage account"
      description: "Legacy account pending migration — INFRA-9876."
```
    {{< /api-field >}}
    {{< api-field name="scope" type="String" required="true" desc="Azure resource scope at which the exemption applies." >}}
```yaml
spec:
  exemptions:
    - displayName: "Exempt legacy storage account"
      scope: >-
        /subscriptions/00000000-0000-0000-0000-000000000000
        /resourceGroups/legacy-rg/providers/Microsoft.Storage
        /storageAccounts/legacystore001
```
    {{< /api-field >}}
    {{< api-field name="exemptionCategory" type="String" default="Waiver" enum="Waiver|Mitigated" desc="Category of the exemption. Waiver is for deliberate policy exceptions. Mitigated is for resources where the policy intent is satisfied through an alternative control." >}}
```yaml
spec:
  exemptions:
    - displayName: "Exempt legacy storage account"
      exemptionCategory: Waiver
```
    {{< /api-field >}}
    {{< api-field name="expiresOn" type="String" desc="Expiration date and time of the exemption in UTC ISO 8601 format (e.g. 2026-12-31T00:00:00Z). After this time the exemption no longer applies." >}}
```yaml
spec:
  exemptions:
    - displayName: "Exempt legacy storage account"
      expiresOn: "2026-06-30T00:00:00Z"
```
    {{< /api-field >}}
  {{< /api-field >}}

{{< /api-field >}}

{{< /api-schema >}}

{{< api-examples >}}

### Assign a CR-referenced policy to a subscription with parameters

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: assign-require-https-storage
  namespace: platform-policies
spec:
  displayName: "Require HTTPS on Storage Accounts"
  description: "Assigns the HTTPS storage policy to the production subscription."
  policyDefinitionRef: require-https-storage
  scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
  enforcementMode: Default
  parameters:
    effect:
      value: "Deny"
  nonComplianceMessages:
    default: "Storage accounts must enforce HTTPS-only traffic. See INFRA-100."
```

### Assign a built-in initiative at management group scope in audit mode

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: assign-cis-benchmark
  namespace: platform-policies
spec:
  displayName: "CIS Azure Benchmark (Audit)"
  description: "Assigns the CIS Azure Benchmark initiative in audit-only mode across the management group."
  policyDefinitionId: >-
    /providers/Microsoft.Authorization/policySetDefinitions/1f3afdf9-d0c9-4c3d-847f-89da613e70a8
  scope: "/providers/Microsoft.Management/managementGroups/corp-management-group"
  enforcementMode: DoNotEnforce
  notScopes:
    - "/providers/Microsoft.Management/managementGroups/sandbox-management-group"
```

### Assign a CR-referenced policy at management group scope with notScopes

Assign the `require-tag-on-resources` definition across an entire management group, pass the `tagName` parameter, and exclude a specific subscription from evaluation.

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyAssignment
metadata:
  labels:
    app.kubernetes.io/name: policy-operator
    app.kubernetes.io/managed-by: kustomize
  name: require-tags-on-rgs
spec:
  displayName: "Require CostCenter tag on Resource Groups"
  description: "Enforces that all resource groups have a CostCenter tag."
  policyDefinitionRef: require-tag-on-resources
  scope: "/providers/Microsoft.Management/managementGroups/platform-management-group"
  enforcementMode: Default
  notScopes:
    - "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg-legacy"
  parameters:
    tagName:
      value: "CostCenter"
  metadata:
    assignedBy: "platform-team"
    category: "Tags"
```

### Assignment with SystemAssigned identity for deployIfNotExists remediation

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: assign-deploy-diagnostics
  namespace: platform-policies
spec:
  displayName: "Deploy Diagnostic Settings for Storage Accounts"
  policyDefinitionRef: deploy-storage-diagnostics
  scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
  enforcementMode: Default
  parameters:
    logAnalyticsWorkspaceId:
      value: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/monitoring-rg/providers/Microsoft.OperationalInsights/workspaces/central-logs"
  identity:
    type: SystemAssigned
    location: australiaeast
    permissions:
      - role: Contributor
        scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
```

### Assignment with resource selectors, per-policy messages, and an inline exemption

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: assign-security-baseline-partial
  namespace: platform-policies
spec:
  displayName: "Security Baseline (Storage — Phased Rollout)"
  policyDefinitionRef: security-baseline-initiative
  scope: "/subscriptions/00000000-0000-0000-0000-000000000000"
  enforcementMode: Default
  parameters:
    storageEffect:
      value: "Audit"
  resourceSelectors:
    - name: storage-accounts-only
      selectors:
        - property: resourceType
          operator: In
          values: ["Microsoft.Storage/storageAccounts"]
        - property: resourceLocation
          operator: In
          values: ["australiaeast", "australiasoutheast"]
  nonComplianceMessages:
    default: "This resource does not meet the platform security baseline."
    perPolicy:
      - policyReferenceId: RequireHttpsOnStorage
        message: "Storage accounts must enforce HTTPS-only traffic."
  exemptions:
    - displayName: "Exempt legacy storage account"
      description: "Legacy account pending migration — INFRA-9876."
      scope: >-
        /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/legacy-rg/providers/Microsoft.Storage/storageAccounts/legacystore001
      exemptionCategory: Waiver
      expiresOn: "2026-06-30T00:00:00Z"
```

### Import an existing Azure Policy Assignment (reconcile)

Adopt and continuously reconcile an existing assignment. The `import-name` annotation preserves the original Azure assignment name so the operator does not create a duplicate when the Kubernetes resource name differs.

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyAssignment
metadata:
  labels:
    app.kubernetes.io/name: policy-operator
    app.kubernetes.io/managed-by: kustomize
  name: audit-vms-dr
  annotations:
    governance.platform.io/import-id: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/microsoft.authorization/policyassignments/785fcbc8d4df43f6a63ac030"
    governance.platform.io/import-mode: "reconcile"
    governance.platform.io/import-name: "785fcbc8d4df43f6a63ac030"
spec:
  displayName: "Policy operator: Require Cost Center tag on Resource Groups - imported"
  description: "Enforces that all resource groups have a CostCenter tag."
  policyDefinitionId: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/Microsoft.Authorization/policyDefinitions/4c47f444f5d44a1db20f8e36"
  parameters:
    allowedLocations:
      value:
        - "eastus2"
  scope: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018"
  notScopes:
    - "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/resourceGroups/rg-taj"
  enforcementMode: Default
  identity:
    type: SystemAssigned
    location: westeurope
```

### Import an existing assignment and bind to a policyDefinitionRef

Import an existing Azure assignment while referencing the policy definition as a CR (`policyDefinitionRef`) instead of a hard-coded Azure resource ID. The operator resolves the CR name to the Azure definition ID at reconcile time.

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: assign-require-cost-center-tag
  namespace: platform-policies
  annotations:
    governance.platform.io/import-id: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/microsoft.authorization/policyassignments/a1b2c3d4e5f643a1bc2d3e4f5a6b7c8d"
    governance.platform.io/import-mode: "observe-only"
    governance.platform.io/import-name: "a1b2c3d4e5f643a1bc2d3e4f5a6b7c8d"
spec:
  displayName: "Require CostCenter tag on Resource Groups"
  description: "Assigns the CostCenter tag policy to the production subscription via CR reference."
  policyDefinitionRef: require-cost-center-tag
  scope: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018"
  enforcementMode: Default
  notScopes:
    - "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/resourceGroups/rg-legacy"
  identity:
    type: SystemAssigned
    location: westeurope
```

{{< /api-examples >}}

{{< api-status >}}

{{< api-field name="assignmentId" type="String" desc="Full Azure resource ID of the created or updated policy assignment, populated after the first successful reconcile." >}}
```yaml
status:
  assignmentId: >-
    /subscriptions/00000000-0000-0000-0000-000000000000
    /providers/Microsoft.Authorization/policyAssignments/assign-security-baseline
```
{{< /api-field >}}

{{< api-field name="assignedLocation" type="String" desc="Azure location set on the policy assignment. Populated when a managed identity is used and persisted for updates." >}}
```yaml
status:
  assignedLocation: "australiaeast"
```
{{< /api-field >}}

{{< api-field name="miPrincipalId" type="String" desc="Principal ID of the managed identity associated with this assignment. Present when spec.identity.type is SystemAssigned or UserAssigned." >}}
```yaml
status:
  miPrincipalId: "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
```
{{< /api-field >}}

{{< api-field name="exemptions" type="Array" children="true" desc="Tracks Azure resource IDs of inline exemptions created for this assignment." >}}
  {{< api-field name="displayName" type="String" desc="Display name matching the exemption spec entry." >}}
```yaml
status:
  exemptions:
    - displayName: "Exempt legacy storage account"
```
  {{< /api-field >}}
  {{< api-field name="exemptionId" type="String" desc="Full Azure resource ID of the created exemption." >}}
```yaml
status:
  exemptions:
    - exemptionId: >-
        /subscriptions/00000000-0000-0000-0000-000000000000
        /providers/Microsoft.Authorization/policyExemptions/legacy-store-waiver
```
  {{< /api-field >}}
  {{< api-field name="scope" type="String" desc="Azure resource scope of the exemption. Stored by the operator for deletion during cleanup." >}}
```yaml
status:
  exemptions:
    - scope: >-
        /subscriptions/00000000-0000-0000-0000-000000000000
        /resourceGroups/legacy-rg
```
  {{< /api-field >}}
{{< /api-field >}}

{{< api-field name="conditions" type="Array" desc="Standard Kubernetes conditions reflecting the current reconcile state." >}}

| Type | Status | Reason | Meaning |
|---|---|---|---|
| `Ready` | `True` | `ReconcileSucceeded` | Assignment is in sync with Azure |
| `Ready` | `False` | `ReconcileFailed` | Last reconcile failed; see message |
| `Ready` | `False` | `AzureAPIError` | ARM API returned an error |

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      message: "Policy assignment successfully reconciled"
      lastTransitionTime: "2024-01-15T10:30:00Z"
```
{{< /api-field >}}

{{< /api-status >}}

{{< api-tips >}}

<div class="tips-list">

  <div class="tips-item">
    <button class="tips-item__header" data-api-toggle aria-expanded="false" aria-controls="tip-import-assignments" type="button">
      Importing Azure Policy Assignments
      <svg class="tips-item__chevron" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="4 6 8 10 12 6"/></svg>
    </button>
    <div class="tips-item__body" id="tip-import-assignments" hidden>
      <p>If you already have policy assignments running in Azure, you don't need to recreate them from scratch. Export the existing assignment with the Azure CLI and hand it off to your AI assistant to generate the manifest — it will fill in <code>policyDefinitionId</code>, <code>parameters</code>, <code>scope</code>, <code>notScopes</code>, and <code>identity</code> while you do something more productive.</p>
<div class="highlight"><pre><code class="language-bash"># Export an existing assignment — pipe to your AI assistant or save for reference
az policy assignment show \
  --name "785fcbc8d4df43f6a63ac030" \
  --scope "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018"</code></pre></div>
      <h3>Always start with <code>observe-only</code></h3>
      <p>Before allowing the operator to make any changes, import with <code>observe-only</code> mode. The operator fetches the live assignment from Azure and populates <code>status</code> — nothing in Azure is touched. Use this to validate that your spec matches the live state before committing to full reconciliation.</p>
<div class="highlight"><pre><code class="language-yaml">metadata:
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/microsoft.authorization/policyassignments/785fcbc8d4df43f6a63ac030
    governance.platform.io/import-name: "785fcbc8d4df43f6a63ac030"
    governance.platform.io/import-mode: "observe-only"</code></pre></div>
      <h3>The <code>import-name</code> annotation is critical</h3>
      <p>Azure policy assignment names are often opaque GUIDs like <code>785fcbc8d4df43f6a63ac030</code>. Your Kubernetes resource name will typically be something human-readable like <code>audit-vms-dr</code>. The <code>import-name</code> annotation bridges this gap — it tells the operator which Azure assignment name to use when calling the ARM API, preventing it from creating a duplicate with the Kubernetes resource name.</p>
<div class="highlight"><pre><code class="language-yaml">metadata:
  name: audit-vms-dr                         # human-readable Kubernetes name
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/.../policyassignments/785fcbc8d4df43f6a63ac030
    governance.platform.io/import-name: "785fcbc8d4df43f6a63ac030"  # actual Azure name</code></pre></div>
      <h3>Check status and conditions with kubectl</h3>
      <p>After deploying, inspect what the operator sees:</p>
<div class="highlight"><pre><code class="language-bash">kubectl describe azurepolicyassignment audit-vms-dr
kubectl get azurepolicyassignment audit-vms-dr -o yaml</code></pre></div>
      <p>The <code>status.assignmentId</code> is populated once the operator has successfully adopted the assignment. The <code>Ready</code> condition reflects whether the last reconcile succeeded.</p>
      <h3>When you're ready, switch to <code>reconcile</code></h3>
      <p>Once you've verified that the operator has correctly adopted the assignment and the spec reflects the desired state, switch to <strong><code>reconcile</code></strong> for continuous GitOps management. Or use <strong><code>adopt-once</code></strong> to apply your spec a single time and then step back.</p>
<div class="highlight"><pre><code class="language-yaml">governance.platform.io/import-mode: "reconcile"   # continuous sync
governance.platform.io/import-mode: "adopt-once"  # adopt once, then hands off</code></pre></div>
      <div class="callout callout-warning" style="margin-top:1.25rem">
        <div class="callout-title">Never remove import annotations</div>
        <p>The operator uses <code>governance.platform.io/import-name</code> as the Azure assignment name instead of the Kubernetes resource name. Removing this annotation causes the operator to fall back to the CRD <code>metadata.name</code>, losing track of the original assignment and potentially creating a duplicate in Azure. Keep both <code>import-id</code> and <code>import-name</code> annotations in place for the lifetime of the resource.</p>
      </div>
    </div>
  </div>

</div>

{{< /api-tips >}}
