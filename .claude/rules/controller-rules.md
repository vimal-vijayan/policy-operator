---
paths:
  - "interal/controller/**"
---

# Controller Rules for Azure Policy Operator
- use the folder structure under internal/controller to organize code by resource type (e.g., definitions, initiatives, assignments, exemptions)
- use the client folder under internal/controller for all interactions with the Azure SDK for Go to manage Azure Policy resources
- use the service folder under internal/controller for business logic related to reconciling Kubernetes CRs with Azure Policy resources
- ensure that all controller code follows the Kubernetes Operator pattern and best practices for writing controllers with Kubebuilder