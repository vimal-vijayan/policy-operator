# policy-operator

A Kubernetes Operator built with [Kubebuilder](https://book.kubebuilder.io/) that manages **Azure Policy resources** through Kubernetes Custom Resources.

Platform teams can define Azure governance policies as Kubernetes manifests, and the operator translates them into Azure Policy API calls via the Azure Resource Manager (ARM) REST API.

## Supported Resources

| Kind | Azure Resource |
|---|---|
| `AzurePolicyDefinition` | Policy Definition |
| `AzurePolicyInitiative` | Policy Set Definition |
| `AzurePolicyAssignment` | Policy Assignment |
| `AzurePolicyExemption` | Policy Exemption |
| `AzurePolicyRemediation` | Policy Remediation |

All resources belong to the `governance.platform.io/v1alpha1` API group.

## Description

The operator watches Custom Resources in a Kubernetes cluster and reconciles them against the Azure Policy API. Each CR maps closely to the Azure Policy REST API model, with additional conveniences such as:

- **Cross-resource references** — use `policyDefinitionRef` or `policyAssignmentRef` to reference other CRs by name instead of hardcoding Azure resource IDs.
- **Managed identity support** — configure `SystemAssigned` or `UserAssigned` identities with automatic role assignments for `deployIfNotExists`/`modify` policies.
- **Inline exemptions** — define exemptions directly on an `AzurePolicyAssignment` without creating a separate `AzurePolicyExemption` CR.
- **Semver versioning** — set `spec.version` on definitions and initiatives; the operator injects it into Azure Policy metadata automatically.
- **Flexible policy rule input** — supply `policyRule` as a structured YAML object or `policyRuleJson` as a raw JSON string.

## Getting Started

### Prerequisites

- go v1.22.0+
- docker 17.03+
- kubectl v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster
- An Azure subscription or management group with sufficient permissions to create policy resources

### Quick Example

**Define a policy:**

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: require-tag-on-resources
spec:
  displayName: "Require a tag on resources"
  description: "Enforces the existence of a required tag on all resources."
  policyType: Custom
  mode: Indexed
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
  managementGroupId: "my-management-group"
```

**Assign the policy:**

```yaml
apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: require-tags-on-rgs
spec:
  displayName: "Require CostCenter tag on Resource Groups"
  policyDefinitionRef: require-tag-on-resources
  scope: "/providers/Microsoft.Management/managementGroups/my-management-group"
  enforcementMode: Default
  parameters:
    tagName: "CostCenter"
```

## Deploy

### Build and push the operator image

```sh
make docker-build docker-push IMG=<some-registry>/policy-operator:tag
```

### Install CRDs into the cluster

```sh
make install
```

### Deploy the operator

```sh
make deploy IMG=<some-registry>/policy-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need cluster-admin privileges.

### Apply sample resources

```sh
kubectl apply -k config/samples/
```

## Uninstall

```sh
# Delete sample CRs
kubectl delete -k config/samples/

# Remove CRDs
make uninstall

# Remove the operator
make undeploy
```

## Distribution

Build a single `install.yaml` bundle for distribution:

```sh
make build-installer IMG=<some-registry>/policy-operator:tag
```

This generates `dist/install.yaml` containing all resources built with Kustomize. Users can then install without cloning the repo:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/policy-operator/<tag>/dist/install.yaml
```

## Development

```sh
# Run locally against the cluster in your current kubeconfig context
make run

# Run tests
make test

# Format and lint
go fmt ./...
go vet ./...
```

Run `make help` for all available targets.

## Contributing

Please open an issue or pull request. Ensure all code passes `go fmt` and `go vet` before submitting.

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
