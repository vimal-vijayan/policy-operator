# Argo CD Deployment for Azure Policy Manifests

This guide shows a recommended repository layout and an `ApplicationSet` that creates one Argo CD `Application` per declared app-root folder for:

- assignments
- exemptions
- remediations

## Recommended Folder Structure

```text
policies/
├── definitions/
│   ├── def-allowed-locations.yaml
│   ├── def-tag-enforcement.yaml
│   └── def-deny-public-ip.yaml
├── assignments/
│   ├── landingzones/
│   │   ├── assign-allowed-locations.yaml
│   │   └── assign-required-tags.yaml
│   ├── migrations/
│   │   └── assign-migration-guardrails.yaml
│   └── management/
│       └── assign-management-baseline.yaml
├── exemptions/
│   ├── landingzones/
│   │   └── exempt-lz-app1-temp.yaml
│   ├── migrations/
│   │   └── exempt-mig-temp-window.yaml
│   └── management/
│       └── exempt-mgmt-breakglass.yaml
└── remediations/
    ├── landingzones/
    │   └── rem-lz-tag-enforcement.yaml
    ├── migrations/
    │   └── rem-mig-legacy-cleanup.yaml
    └── management/
        └── rem-mgmt-baseline-fix.yaml
```

## ApplicationSet (Depth-Based Apps)

If you do not want marker files in the policy repository, use a depth-based split instead:

```text
assignments/
└── essity/
  ├── direct-file.yaml                     -> assi-essity
  ├── essity-migration/
  │   ├── direct-file.yaml                 -> assi-essity-migration
  │   └── sandbox/
  │       └── nested-file.yaml             -> assi-essity-migration-sandbox
  ├── landingzones/
  │   ├── direct-file.yaml                 -> assi-landingzones
  │   ├── corporate/
  │   │   └── nested-file.yaml             -> assi-landingzones-corporate
  │   └── online/
  │       └── nested-file.yaml             -> assi-landingzones-online
  ├── platform/
  │   ├── direct-file.yaml                 -> assi-platform
  │   ├── connectivity/
  │   ├── identity/
  │   ├── management/
  │   └── security/
  └── sandboxes/
```

This model uses three layers:

- `assignments/*` and `exemptions/*`: one app per top-level folder, `recurse: false`
- `assignments/*/*` and `exemptions/*/*`: one app per second-level folder, `recurse: false`
- `assignments/*/*/*` and `exemptions/*/*/*`: one app per third-level folder, `recurse: true`

That gives you:

- `assi-essity` for YAML directly under `assignments/essity`
- `assi-landingzones` for YAML directly under `assignments/essity/landingzones`
- `assi-landingzones-corporate` for everything under `assignments/essity/landingzones/corporate`
- `assi-landingzones-online` for everything under `assignments/essity/landingzones/online`

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
        files:
          - path: policies/assignments/**/.argocd-root.yaml
          - path: policies/exemptions/**/.argocd-root.yaml
          - path: policies/remediations/**/.argocd-root.yaml
      template:
        metadata:
          name: '{{ .appName }}'
          labels:
            directories:
              - path: assignments/*

        spec:
              name: 'assi-{{ .path.basename }}'
            directory:
              recurse: false

          destination:
            server: https://kubernetes.default.svc
            namespace: policy-operator-system
                path: '{{ .path.path }}'
          syncPolicy:
            automated:
              prune: true
              selfHeal: true
            syncOptions:
              - CreateNamespace=true

    - git:
        repoURL: https://github.com/your-org/your-repo.git
        # Add equivalent second-level and third-level generators as needed.
## Recommended Argo CD App Split

    Tradeoff: this avoids marker files, but it can create empty apps for folders that exist only to group child folders and have no direct YAML manifests.
- Use the `ApplicationSet` above for assignments, exemptions, and remediations.

This keeps definitions synchronized first and makes operational ownership cleaner by declared app-root folder.
