# Argo CD Deployment for Azure Policy Manifests

This guide shows a recommended repository layout and an `ApplicationSet` that creates one Argo CD `Application` per management-group subfolder for:

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

## ApplicationSet (Per Subfolder App)

This ApplicationSet scans all subfolders under assignments, exemptions, and remediations and creates one application for each folder.

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
      # Examples: assignments-landingzones, assignments-landingzones-corp, exemptions-management
      # Deeper paths are capped at three segments after 'policies'.
      name: '{{ if ge (len .path.segments) 4 }}{{ join "-" (slice .path.segments 1 4) }}{{ else }}{{ join "-" (slice .path.segments 1) }}{{ end }}'
      labels:
        category: policy
        type: '{{ index .path.segments 1 }}'
        managementGroup: '{{ .path.basename }}'

    spec:
      project: default
      source:
        repoURL: https://github.com/your-org/your-repo.git
        targetRevision: main
        path: '{{ .path.path }}'
        directory:
          recurse: true

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

## Recommended Argo CD App Split

- Keep `policies/definitions` in a separate `Application`.
- Use the `ApplicationSet` above for assignments, exemptions, and remediations.

This keeps definitions synchronized first and makes operational ownership cleaner by management-group folder.
