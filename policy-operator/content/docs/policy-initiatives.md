---
title: "Policy Initiatives"
description: "Group related Azure Policy definitions into initiatives (policy sets) using AzurePolicyInitiative."
weight: 20
---

The `AzurePolicyInitiative` custom resource maps to an [Azure Policy Set Definition](https://learn.microsoft.com/en-us/azure/governance/policy/concepts/initiative-definition-structure). An initiative groups one or more policy definitions so they can be assigned and tracked together. The operator reconciles each resource by creating or updating the corresponding policy set definition in Azure.

{{< api-schema kind="AzurePolicyInitiative" version="v1alpha1" examples="5" status="true" tips="true" >}}

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
  {{< api-field name="annotations" type="Object" children="true" desc="Arbitrary non-identifying metadata. The following annotations control import behaviour when adopting an existing Azure Policy Initiative." >}}
    {{< api-field name="governance.platform.io/import-id" type="String" desc="Full Azure resource ID of the existing policy set definition to adopt. Required when importing. Supports both subscription and management group scoped IDs." >}}
```yaml
metadata:
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policySetDefinitions/my-initiative
```
    {{< /api-field >}}
    {{< api-field name="governance.platform.io/import-name" type="String" desc="The bare Azure policy set definition name (last segment of the resource ID). Used as the initiative name when writing back to Azure in reconcile or adopt-once mode." >}}
```yaml
metadata:
  annotations:
    governance.platform.io/import-name: "my-initiative"
```
    {{< /api-field >}}
    {{< api-field name="governance.platform.io/import-mode" type="String" default="observe-only" enum="observe-only|reconcile|adopt-once" desc="Controls what happens after the initiative is imported. observe-only: read-only, no changes are pushed to Azure. reconcile: continuously sync Azure to match the spec on every reconcile. adopt-once: adopt and apply changes once, then stop reconciling." >}}
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

### Import existing initiative (reconcile)

Adopt an existing Azure Policy Initiative and continuously sync Azure to match the spec. Uses a mix of `policyDefinitionRef` (CR-managed definition) and `policyDefinitionId` (externally managed definition) to show both reference styles.

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyInitiative
metadata:
  labels:
    app.kubernetes.io/name: policy-operator
    app.kubernetes.io/managed-by: kustomize
  name: tag-governance-initiative
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/Microsoft.Authorization/policySetDefinitions/41939cde4b42430bbb43d66e
    governance.platform.io/import-name: "41939cde4b42430bbb43d66e"
    governance.platform.io/import-mode: "reconcile"
    # governance.platform.io/import-mode: "observe-only"
    # governance.platform.io/import-mode: "adopt-once"
spec:
  displayName: "Tag Governance Initiative"
  description: "Imported initiative sample for tag governance controls."
  version: "1.0.0"
  metadata:
    category: "platform"
  policyDefinitions:
    - policyDefinitionRef: "require-tag-on-resources"
      parameters:
        tagName:
          value: "costcenter"
    - policyDefinitionId: >-
        /subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/Microsoft.Authorization/policyDefinitions/require-tag-on-resources-with-json
      parameters:
        tagName:
          value: "costcenter"
  subscriptionId: "f2024049-e6cb-4489-9270-6d0d6cd65018"
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
| `Ready` | `True` | `Reconciled` | Initiative is in sync with Azure |
| `Ready` | `True` | `ObservedOnly` | Imported in observe-only mode; no changes pushed to Azure |
| `Ready` | `False` | `ReconcileFailed` | Last reconcile failed; see message |
| `Ready` | `False` | `DeleteFailed` | Finalizer cleanup failed; see message |
| `Ready` | `False` | `ImportFailed` | Import from Azure failed; see message |
| `Ready` | `False` | `ImportConflict` | `import-id` annotation conflicts with already-bound `initiativeId` |
| `Imported` | `True` | `ImportSucceeded` | Existing Azure initiative was adopted successfully |
| `Imported` | `False` | `ImportFailed` | Could not fetch or validate the initiative from Azure |
| `DriftDetected` | `True` | `SpecMismatch` | Live Azure initiative differs from the desired spec; see message for fields |
| `DriftDetected` | `False` | `InSync` | Azure initiative matches the desired spec |

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      reason: Reconciled
      message: "Policy initiative successfully reconciled"
      lastTransitionTime: "2024-01-15T10:30:00Z"
    - type: Imported
      status: "True"
      reason: ImportSucceeded
      message: "Existing Azure Policy Initiative was adopted successfully."
      lastTransitionTime: "2024-01-15T10:30:00Z"
    - type: DriftDetected
      status: "True"
      reason: SpecMismatch
      message: "Live Azure initiative differs from desired spec: displayName, description"
      lastTransitionTime: "2024-01-15T10:30:00Z"
```
{{< /api-field >}}

{{< /api-status >}}

{{< api-tips >}}

<div class="tips-list">

  <div class="tips-item">
    <button class="tips-item__header" data-api-toggle aria-expanded="false" aria-controls="tip-import-initiatives" type="button">
      Importing Azure Policy Initiatives
      <svg class="tips-item__chevron" viewBox="0 0 16 16" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><polyline points="4 6 8 10 12 6"/></svg>
    </button>
    <div class="tips-item__body" id="tip-import-initiatives" hidden>
      <p>Oh, you've got an existing initiative with <em>twelve</em> policy definitions already deployed in Azure. How charming. Manually rewriting all of that into YAML is absolutely how you planned to spend your afternoon. Fear not — your AI assistant has a pulse and can read JSON. Export the initiative, paste it in, and let the machine handle the tedious part while you contemplate your life choices.</p>
      <p>Export your existing initiative and hand it to your AI assistant to generate the <code>AzurePolicyInitiative</code> manifest. It will sort out <code>policyDefinitions</code>, <code>parameters</code>, and all the nested fun:</p>
<div class="highlight"><pre><code class="language-bash"># Export the initiative from Azure CLI, let AI do the YAML gymnastics
az policy set-definition show --name "my-initiative" --subscription "00000000-0000-0000-0000-000000000000"</code></pre></div>
      <h3>Start with <code>observe-only</code> — unless you enjoy surprises</h3>
      <p>Before you let the operator start overwriting things in Azure, import with <code>observe-only</code> mode. It reads the live initiative and reports drift without touching a single resource. Think of it as reading someone's diary before deciding whether to burn it.</p>
<div class="highlight"><pre><code class="language-yaml">metadata:
  annotations:
    governance.platform.io/import-id: >-
      /subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policySetDefinitions/my-initiative
    governance.platform.io/import-name: "my-initiative"
    governance.platform.io/import-mode: "observe-only"</code></pre></div>
      <h3>Watch status and drift with kubectl</h3>
      <p>Once deployed, <code>kubectl</code> will tell you everything the operator knows — which is frankly more than most humans know about their own policies:</p>
<div class="highlight"><pre><code class="language-bash">kubectl describe azurepolicyinitiative tag-governance-initiative
kubectl get azurepolicyinitiative tag-governance-initiative -o yaml</code></pre></div>
      <p>Pay close attention to the <code>DriftDetected</code> condition. If <code>reason: SpecMismatch</code> appears, your spec and the live Azure initiative disagree. You will know exactly which fields are out of sync before you commit to doing anything about it.</p>
      <h3>When you're ready, pick your poison</h3>
      <p>Switch to <strong><code>reconcile</code></strong> for continuous GitOps management — the operator keeps Azure in sync with your spec on every reconcile loop, forever, whether you like it or not. Or use <strong><code>adopt-once</code></strong> to apply your spec exactly once, then quietly walk away.</p>
<div class="highlight"><pre><code class="language-yaml">governance.platform.io/import-mode: "reconcile"   # continuous sync — no escape
governance.platform.io/import-mode: "adopt-once"  # adopt once, then pretend it never happened</code></pre></div>
      <h3>Mix <code>policyDefinitionRef</code> and <code>policyDefinitionId</code> freely</h3>
      <p>Inside <code>policyDefinitions</code>, you can reference CR-managed definitions by name (<code>policyDefinitionRef</code>) or point directly to any Azure resource ID (<code>policyDefinitionId</code>). Mix and match as needed — the operator handles resolution at reconcile time, no handholding required.</p>
<div class="highlight"><pre><code class="language-yaml">policyDefinitions:
  - policyDefinitionRef: "require-tag-on-resources"   # CR in this namespace
    parameters:
      tagName:
        value: "costcenter"
  - policyDefinitionId: >-
      /subscriptions/00000000/providers/Microsoft.Authorization/policyDefinitions/some-external-policy
    parameters:
      tagName:
        value: "costcenter"</code></pre></div>
      <div class="callout callout-warning" style="margin-top:1.25rem">
        <div class="callout-title">Never remove import annotations</div>
        <p>The operator maps the CRD <code>metadata.name</code> to the Azure initiative name by default. Import is the exception — <code>governance.platform.io/import-name</code> is used directly instead. Removing these annotations causes the operator to fall back to the CRD name, losing track of the original initiative and potentially creating a glorious duplicate. Removing import annotations from a managed initiative can produce a policy management experience that nobody asked for.</p>
      </div>
    </div>
  </div>

</div>

{{< /api-tips >}}
