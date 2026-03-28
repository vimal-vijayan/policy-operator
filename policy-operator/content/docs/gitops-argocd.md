---
title: "GitOps with Argo CD"
description: "Deploy and lifecycle-manage Azure Policy manifests through pull requests, code review, and Argo CD sync loops."
weight: 50
---

> **Note:** The architecture and Argo CD deployment model described on this page is a reference design. Actual implementation will vary based on your organisation's specific requirements, tooling, and platform constraints. Treat this as a starting point, not a prescriptive standard.

GitOps treats your Git repository as the single source of truth for cluster state. When combined with the Azure Policy Operator, every Azure Policy definition, assignment, exemption, and remediation becomes a versioned, reviewable, auditable Kubernetes manifest — and Argo CD keeps Azure in sync automatically.

## Why GitOps for Azure governance?

| Without GitOps | With GitOps |
|---|---|
| Policies applied ad-hoc via portal or CLI | Every change goes through pull request + review |
| No record of who changed what, or when | Full audit trail in Git history |
| Drift between environments is invisible | Argo CD detects and optionally heals drift |
| Rollback means manually re-applying old config | Rollback is `git revert` |
| Exemptions silently persist past their purpose | Expiry enforced in YAML, removed via PR |

---

## Repository layout

Separate **definitions** from **assignments / exemptions / remediations**. Definitions are global building blocks; the others are scoped to management-group subfolders, which enables per-group progressive rollouts.

```text
policies/
├── definitions/                         # global — synced by a dedicated App
│   ├── def-allowed-locations.yaml
│   ├── def-tag-enforcement.yaml
│   └── def-deny-public-ip.yaml
│
├── assignments/                         # scoped per management group
│   ├── landingzones/
│   │   ├── assign-allowed-locations.yaml
│   │   └── assign-required-tags.yaml
│   ├── migrations/
│   │   └── assign-migration-guardrails.yaml
│   └── management/
│       └── assign-management-baseline.yaml
│
└── exemptions/
    ├── landingzones/
    │   └── exempt-lz-app1-temp.yaml
    ├── migrations/
    │   └── exempt-mig-temp-window.yaml
    └── management/
        └── exempt-mgmt-breakglass.yaml

```

> **Tip:** Keep definitions in a flat folder. Their names are globally unique and don't need a management-group split. Everything else benefits from per-group isolation.

---

## Argo CD application model

Use **two Argo CD objects** — one `Application` for definitions, and one `ApplicationSet` that auto-generates an `Application` per management-group subfolder.

### 1. Definitions Application

Definitions must be reconciled before assignments reference them. A standalone `Application` with `Automated` sync ensures they land first.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: policy-definitions
  namespace: argocd
spec:
  project: default

  source:
    repoURL: https://github.com/your-org/your-repo.git
    targetRevision: main
    path: policies/definitions
    directory:
      recurse: false          # flat folder — no recursion needed

  destination:
    server: https://kubernetes.default.svc
    namespace: policy-operator-system

  syncPolicy:
    automated:
      prune: true             # remove definitions deleted from Git
      selfHeal: true          # revert out-of-band changes
    syncOptions:
      - CreateNamespace=true
```

### 2. ApplicationSet — one App per management-group folder

The `ApplicationSet` scans every subfolder under `assignments`, `exemptions`, and `remediations` and creates a dedicated Argo CD `Application` for each one. This gives fine-grained sync status, per-group RBAC, and the ability to pause a single management group without affecting others.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: policies-subfolders
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions:
    - missingkey=error

  generators:
    - git:
        repoURL: https://github.com/your-org/your-repo.git
        revision: main
        directories:
          - path: policies/assignments/*
          - path: policies/exemptions/*
          - path: policies/remediations/*

  template:
    metadata:
      # Produces names like: assignments-landingzones, exemptions-management
      name: '{{ index .path.segments 1 }}-{{ .path.basename }}'
      labels:
        category: policy
        type:            '{{ index .path.segments 1 }}'
        managementGroup: '{{ .path.basename }}'

    spec:
      project: default

      source:
        repoURL: https://github.com/your-org/your-repo.git
        targetRevision: main
        path: '{{ .path.path }}'
        directory:
          recurse: true       # pick up any sub-subfolders

      destination:
        server: https://kubernetes.default.svc
        namespace: policy-operator-system

      syncPolicy:
        automated:
          prune: true
          selfHeal: true
        syncOptions:
          - CreateNamespace=true
```

---

## Progressive rollout pattern

Because each management group is its own Argo CD `Application`, you control the blast radius of every change:

```
1. Merge PR → auto-synced to landingzones  (lowest risk)
2. Review compliance dashboard
3. Manually sync migrations
4. Manually sync management  (highest blast radius)
```

To hold a management group back from auto-sync, patch its generated Application:

```bash
# Disable auto-sync for the management group during a freeze
argocd app set assignments-management --sync-policy none
```

Re-enable when ready:

```bash
argocd app set assignments-management --sync-policy automated
```

---

## Multi-environment strategy

Use **branch-per-environment** or **folder-per-environment** depending on how divergent your environments are.

### Branch-per-environment (recommended for strong isolation)

```
main          → production cluster
staging       → staging cluster
dev           → dev cluster
```

Point each Argo CD Application's `targetRevision` to the appropriate branch. Promote by raising a PR from `dev → staging → main`.

### Folder-per-environment (recommended for shared clusters)

```
policies/
├── base/                # shared definitions
├── overlays/
│   ├── dev/
│   ├── staging/
│   └── production/
```

Use Kustomize overlays to patch scope and effect per environment:

```yaml
# overlays/dev/kustomization.yaml
resources:
  - ../../base
patches:
  - path: patch-audit-only.yaml   # override Deny → Audit in dev
```

---

## Drift detection and self-healing

With `selfHeal: true`, Argo CD continuously compares the live cluster state against Git. If someone applies a manifest directly (bypassing Git), Argo CD reverts it within the next sync cycle (default: 3 minutes).

Check live drift at any time:

```bash
argocd app diff assignments-landingzones
```

View a full sync status summary:

```bash
argocd app list -l category=policy
```

---

## Secrets and credential management

The Azure Policy Operator needs an Azure identity (Service Principal or Workload Identity) to call the ARM API. Keep credentials out of Git:

### Option A — Kubernetes Secret (pre-created)

```bash
kubectl create secret generic azure-policy-credentials \
  --namespace policy-operator-system \
  --from-literal=clientId=<CLIENT_ID> \
  --from-literal=clientSecret=<CLIENT_SECRET> \
  --from-literal=tenantId=<TENANT_ID>
```

Reference in your Helm values (not in Git):

```yaml
# argocd/policy-operator-app.yaml — values passed via Argo CD UI/CLI
helm:
  parameters:
    - name: azure.existingSecret
      value: azure-policy-credentials
```

### Option B — Workload Identity (recommended for AKS)

Annotate the operator's service account with the managed identity client ID. No secret required in the cluster:

```yaml
serviceAccount:
  annotations:
    azure.workload.identity/client-id: "<MANAGED_IDENTITY_CLIENT_ID>"
```

### Option C — External Secrets Operator

Sync credentials from Azure Key Vault into a Kubernetes Secret automatically:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: azure-policy-credentials
  namespace: policy-operator-system
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: azure-keyvault
    kind: ClusterSecretStore
  target:
    name: azure-policy-credentials
  data:
    - secretKey: clientSecret
      remoteRef:
        key: policy-operator-client-secret
```

---

## Example policy manifests

### Definition — Deny public IP

```yaml
# policies/definitions/def-deny-public-ip.yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: def-deny-public-ip
spec:
  displayName: "Deny public IP addresses"
  description: "Prevents creation of public IP resources across all managed scopes."
  policyType: Custom
  mode: All
  policyRule:
    if:
      field: "type"
      equals: "Microsoft.Network/publicIPAddresses"
    then:
      effect: Deny
```

### Assignment — Landing zones scope

```yaml
# policies/assignments/landingzones/assign-deny-public-ip.yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: assign-deny-public-ip-lz
spec:
  displayName: "Deny public IPs — Landing Zones"
  policyDefinitionRef:
    name: def-deny-public-ip
  scope: /providers/Microsoft.Management/managementGroups/landingzones
  enforcementMode: Default
```

### Time-bound exemption

```yaml
# policies/exemptions/landingzones/exempt-lz-app1-temp.yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyExemption
metadata:
  name: exempt-lz-app1-public-ip
  annotations:
    policy.azure.com/owner: platform-team
    policy.azure.com/ticket: "OPS-4821"
spec:
  policyAssignmentRef:
    name: assign-deny-public-ip-lz
  exemptionCategory: Waiver
  displayName: "Temporary waiver — App1 migration window"
  expiresOn: "2026-06-30T00:00:00Z"
  resourceSelectors:
    - name: app1-resources
      selectors:
        - kind: ResourceGroup
          in:
            - app1-migration-rg
```

---

## Recommended Argo CD project setup

Isolate policy workloads into a dedicated Argo CD project with scoped permissions:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: azure-policy
  namespace: argocd
spec:
  description: "Azure Policy GitOps managed by Azure Policy Operator"

  sourceRepos:
    - https://github.com/your-org/your-repo.git

  destinations:
    - namespace: policy-operator-system
      server: https://kubernetes.default.svc

  clusterResourceWhitelist:
    - group: policy.azure.com
      kind: AzurePolicyDefinition
    - group: policy.azure.com
      kind: AzurePolicyAssignment
    - group: policy.azure.com
      kind: AzurePolicyExemption
    - group: policy.azure.com
      kind: AzurePolicyRemediation

  roles:
    - name: policy-admin
      description: Full access to policy applications
      policies:
        - p, proj:azure-policy:policy-admin, applications, *, azure-policy/*, allow
    - name: policy-viewer
      description: Read-only access
      policies:
        - p, proj:azure-policy:policy-viewer, applications, get, azure-policy/*, allow
```
