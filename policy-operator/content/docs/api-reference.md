---
title: "API Reference"
description: "Full specification for all Policy Operator Custom Resource Definitions."
weight: 3
---

## Supported Kinds

The operator manages the following custom resources:

| Kind | API Version | Description |
|------|-------------|-------------|
| `AzurePolicyDefinition` | `policy.azure.com/v1alpha1` | Azure Policy definition |
| `AzurePolicyInitiative` | `policy.azure.com/v1alpha1` | Policy set definition |
| `AzurePolicyAssignment` | `policy.azure.com/v1alpha1` | Policy / initiative assignment |
| `AzurePolicyExemption` | `policy.azure.com/v1alpha1` | Policy exemption |
| `AzurePolicyRemediation` | `policy.azure.com/v1alpha1` | Remediation task |

## AzurePolicyDefinition

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: my-policy
spec:
  displayName: string          # Human-readable name
  description: string          # Optional description
  policyType: Custom | BuiltIn # Default: Custom
  mode: All | Indexed          # Default: All
  metadata: {}                 # Arbitrary metadata map
  parameters: {}               # Parameter definitions
  policyRule:                  # Required — the policy rule object
    if: {}
    then: {}
```

## AzurePolicyInitiative

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyInitiative
metadata:
  name: my-initiative
spec:
  displayName: string
  description: string
  metadata: {}
  parameters: {}
  policyDefinitions:
    - policyDefinitionId: string    # ARM resource ID or CRD name ref
      parameters: {}
```

## AzurePolicyAssignment

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: my-assignment
spec:
  displayName: string
  scope: string               # Management group / subscription / resource group ID
  policyDefinitionId: string  # ARM ID of the definition or initiative
  parameters: {}
  enforcementMode: Default | DoNotEnforce
  notScopes: []               # List of resource IDs to exclude
```

## AzurePolicyExemption

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyExemption
metadata:
  name: my-exemption
spec:
  scope: string
  policyAssignmentId: string
  exemptionCategory: Waiver | Mitigated
  expiresOn: "2026-12-31T00:00:00Z"   # RFC 3339, optional
  description: string
```

## AzurePolicyRemediation

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyRemediation
metadata:
  name: my-remediation
spec:
  scope: string
  policyAssignmentId: string
  policyDefinitionReferenceId: string  # For initiative assignments
  resourceDiscoveryMode: ExistingNonCompliant | ReEvaluateCompliance
  parallelDeployments: 10
  failureThreshold:
    percentage: 0.1
```
