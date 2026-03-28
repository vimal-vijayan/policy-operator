---
title: "Getting Started"
description: "Install and configure the Azure Policy Operator in your Kubernetes cluster."
weight: 2
---

## Prerequisites

Before installing the Azure Policy Operator, ensure you have:

- A Kubernetes cluster (v1.24+)
- `kubectl` configured against your cluster
- An Azure subscription with permissions to create and assign policies
- A Service Principal or Managed Identity with the **Resource Policy Contributor** role

## Installation

### 1. Install with Helm

```bash
helm repo add policy-operator https://charts.example.com/policy-operator
helm repo update

helm install policy-operator policy-operator/policy-operator \
  --namespace policy-system \
  --create-namespace \
  --set azure.tenantId=<TENANT_ID> \
  --set azure.subscriptionId=<SUBSCRIPTION_ID> \
  --set azure.clientId=<CLIENT_ID> \
  --set azure.clientSecret=<CLIENT_SECRET>
```

### 2. Verify installation

```bash
kubectl get pods -n policy-system
# NAME                               READY   STATUS    RESTARTS   AGE
# policy-operator-6d9b4c8f7-xk2lp   1/1     Running   0          30s
```

## Your first policy definition

Create a file `deny-public-ip.yaml`:

```yaml
apiVersion: policy.azure.com/v1alpha1
kind: AzurePolicyDefinition
metadata:
  name: deny-public-ip
spec:
  displayName: "Deny public IP addresses"
  description: "Prevents creation of public IP resources."
  policyType: Custom
  mode: All
  policyRule:
    if:
      field: "type"
      equals: "Microsoft.Network/publicIPAddresses"
    then:
      effect: Deny
```

Apply it:

```bash
kubectl apply -f deny-public-ip.yaml
```

Check status:

```bash
kubectl get azurepolicydefinition deny-public-ip -o yaml
```
