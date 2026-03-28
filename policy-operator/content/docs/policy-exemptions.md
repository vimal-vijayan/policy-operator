---
title: "Policy Exemptions"
description: "Create time-bound or permanent Azure Policy exemptions for resources using AzurePolicyExemption."
weight: 40
---

The `AzurePolicyExemption` custom resource maps to an [Azure Policy Exemption](https://learn.microsoft.com/en-us/azure/governance/policy/concepts/exemption-structure). An exemption excludes a resource scope from evaluation under a specific policy assignment — without removing the assignment itself. Common uses include temporary waivers for legacy resources and mitigated exceptions where an alternative control satisfies the policy intent. The operator reconciles each resource by creating or updating the corresponding exemption in Azure.

{{< api-schema kind="AzurePolicyExemption" version="v1alpha1" examples="4" status="true" >}}

{{< api-field name="apiVersion" type="String" desc="API version for this resource. Must be policy.azure.com/v1alpha1." >}}
```yaml
apiVersion: policy.azure.com/v1alpha1
```
{{< /api-field >}}

{{< api-field name="kind" type="String" desc="Resource kind. Must be AzurePolicyExemption." >}}
```yaml
kind: AzurePolicyExemption
```
{{< /api-field >}}

{{< api-field name="metadata" type="Object" children="true" desc="Standard Kubernetes object metadata." >}}
  {{< api-field name="name" type="String" required="true" desc="Unique name of the resource within its namespace." >}}
```yaml
metadata:
  name: exempt-legacy-storage
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
    policy.azure.com/ticket: "INFRA-9876"
```
  {{< /api-field >}}
{{< /api-field >}}

{{< api-field name="spec" type="Object" required="true" children="true" desc="Desired state of the AzurePolicyExemption." >}}

  {{< api-field name="displayName" type="String" required="true" desc="Human-readable name shown in the Azure portal." >}}
```yaml
spec:
  displayName: "Exempt legacy storage account from HTTPS policy"
```
  {{< /api-field >}}

  {{< api-field name="description" type="String" desc="A human-readable explanation for why this exemption exists." >}}
```yaml
spec:
  description: "Legacy account pending migration to new storage tier — tracked in INFRA-9876."
```
  {{< /api-field >}}

  {{< api-field name="policyAssignmentRef" type="String" required="true" mutual="policyAssignmentId" desc="Name of an AzurePolicyAssignment CR in the same namespace. The operator resolves this reference to the Azure assignment ID from the CR's status at reconcile time. Exactly one of policyAssignmentRef or policyAssignmentId must be specified." >}}
```yaml
spec:
  policyAssignmentRef: assign-require-https-storage
```
  {{< /api-field >}}

  {{< api-field name="policyAssignmentId" type="String" required="true" mutual="policyAssignmentRef" desc="Full Azure resource ID of the policy assignment being exempted. Use this when the assignment is managed outside the operator. Exactly one of policyAssignmentRef or policyAssignmentId must be specified." >}}
```yaml
spec:
  policyAssignmentId: >-
    /subscriptions/00000000-0000-0000-0000-000000000000
    /providers/Microsoft.Authorization/policyAssignments/assign-require-https-storage
```
  {{< /api-field >}}

  {{< api-field name="scope" type="String" required="true" desc="The Azure resource scope at which the exemption applies. Can be a subscription, resource group, or individual resource. Must be at or below the scope of the referenced assignment." >}}
```yaml
spec:
  # Individual resource scope
  scope: >-
    /subscriptions/00000000-0000-0000-0000-000000000000
    /resourceGroups/legacy-rg/providers/Microsoft.Storage
    /storageAccounts/legacystore001
  # Resource group scope
  # scope: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/legacy-rg"
```
  {{< /api-field >}}

  {{< api-field name="exemptionCategory" type="String" default="Waiver" enum="Waiver|Mitigated" desc="Category of the exemption. Waiver is for deliberate policy exceptions where the policy requirement is acknowledged but intentionally bypassed. Mitigated is for resources where the policy intent is satisfied through an alternative control." >}}
```yaml
spec:
  exemptionCategory: Waiver
  # exemptionCategory: Mitigated  # alternative control satisfies the policy intent
```
  {{< /api-field >}}

  {{< api-field name="expiresOn" type="String" desc="Expiration date and time of the exemption in UTC ISO 8601 format (e.g. 2026-12-31T00:00:00Z). After this time Azure no longer applies the exemption and the resource is evaluated against the assignment again. Omit for permanent exemptions." >}}
```yaml
spec:
  expiresOn: "2026-06-30T00:00:00Z"
```
  {{< /api-field >}}

  {{< api-field name="resourceSelectors" type="Array" children="true" desc="Filters the set of resources within the exemption scope that are exempt. Useful for exempting a subset of resources under a resource group scope rather than the entire group." >}}
    {{< api-field name="name" type="String" required="true" desc="Name of the resource selector." >}}
```yaml
spec:
  resourceSelectors:
    - name: legacy-storage-only
```
    {{< /api-field >}}
    {{< api-field name="selectors" type="Array" children="true" desc="List of selector expressions combined with AND logic." >}}
      {{< api-field name="property" type="String" required="true" enum="resourceType|resourceLocation|resourceWithoutLocation|userPrincipalId|groupPrincipalId" desc="The resource property to filter on." >}}
```yaml
spec:
  resourceSelectors:
    - name: legacy-storage-only
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
    - name: australia-only
      selectors:
        - property: resourceLocation
          operator: In
          values: ["australiaeast", "australiasoutheast"]
```
      {{< /api-field >}}
      {{< api-field name="values" type="Array" required="true" desc="List of values to match against the selected property." >}}
```yaml
spec:
  resourceSelectors:
    - name: legacy-storage-only
      selectors:
        - property: resourceType
          operator: In
          values: ["Microsoft.Storage/storageAccounts"]
```
      {{< /api-field >}}
    {{< /api-field >}}
  {{< /api-field >}}

{{< /api-field >}}

{{< /api-schema >}}

{{< api-examples >}}

### Time-bound waiver for a legacy storage account

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyExemption
metadata:
  name: exempt-legacy-storage
  namespace: platform-policies
  annotations:
    policy.azure.com/ticket: "INFRA-9876"
spec:
  displayName: "Exempt legacy storage account from HTTPS policy"
  description: "Legacy account pending migration to new storage tier — INFRA-9876."
  policyAssignmentRef: assign-require-https-storage
  scope: >-
    /subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/legacy-rg/providers/Microsoft.Storage/storageAccounts/legacystore001
  exemptionCategory: Waiver
  expiresOn: "2026-06-30T00:00:00Z"
```

### Mitigated exemption for a resource group with an alternative control

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyExemption
metadata:
  name: exempt-firewall-rg-mitigated
  namespace: platform-policies
spec:
  displayName: "Firewall RG — public IP mitigated by WAF"
  description: "Public IP addresses in this resource group are protected by Azure WAF, satisfying the policy intent."
  policyAssignmentRef: assign-cis-benchmark
  scope: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/firewall-rg"
  exemptionCategory: Mitigated
```

### Exemption referencing an externally managed assignment ID

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyExemption
metadata:
  name: exempt-sandbox-subscription
  namespace: platform-policies
spec:
  displayName: "Exempt sandbox subscription from location policy"
  description: "Sandbox subscriptions are permitted to deploy to any region for testing purposes."
  policyAssignmentId: >-
    /providers/Microsoft.Management/managementGroups/corp-management-group/providers/Microsoft.Authorization/policyAssignments/restrict-locations
  scope: "/subscriptions/11111111-1111-1111-1111-111111111111"
  exemptionCategory: Waiver
  expiresOn: "2026-12-31T00:00:00Z"
```

### Scoped exemption using resource selectors within a resource group

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyExemption
metadata:
  name: exempt-storage-in-legacy-rg
  namespace: platform-policies
  annotations:
    policy.azure.com/ticket: "INFRA-4321"
spec:
  displayName: "Exempt storage accounts in legacy-rg from HTTPS policy"
  description: "All storage accounts in legacy-rg are pending migration and exempt until end of Q2 2026."
  policyAssignmentRef: assign-require-https-storage
  scope: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/legacy-rg"
  exemptionCategory: Waiver
  expiresOn: "2026-06-30T00:00:00Z"
  resourceSelectors:
    - name: storage-accounts-only
      selectors:
        - property: resourceType
          operator: In
          values: ["Microsoft.Storage/storageAccounts"]
```

{{< /api-examples >}}

{{< api-status >}}

{{< api-field name="exemptionId" type="String" desc="Full Azure resource ID of the created or updated policy exemption, populated after the first successful reconcile." >}}
```yaml
status:
  exemptionId: >-
    /subscriptions/00000000-0000-0000-0000-000000000000
    /resourceGroups/legacy-rg/providers/Microsoft.Authorization
    /policyExemptions/exempt-legacy-storage
```
{{< /api-field >}}

{{< api-field name="conditions" type="Array" desc="Standard Kubernetes conditions reflecting the current reconcile state." >}}

| Type | Status | Reason | Meaning |
|---|---|---|---|
| `Ready` | `True` | `ReconcileSucceeded` | Exemption is in sync with Azure |
| `Ready` | `False` | `ReconcileFailed` | Last reconcile failed; see message |
| `Ready` | `False` | `AzureAPIError` | ARM API returned an error |

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: ReconcileSucceeded
      message: "Policy exemption successfully reconciled"
      lastTransitionTime: "2024-01-15T10:30:00Z"
```
{{< /api-field >}}

{{< /api-status >}}
