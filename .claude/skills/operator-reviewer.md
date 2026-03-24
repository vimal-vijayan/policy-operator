---
name: operator-reviewer
description: This skill focuses on reviewing the implementation of the Kubernetes operator for Azure Policy management. It includes ensuring that the operator logic correctly interacts with the Azure API, that the CRD definitions are accurately structured, and that the overall design follows best practices for Kubernetes operators and Azure resource management.
---

## Purpose
Used to review the implementation of the Kubernetes operator for Azure Policy management. This includes reviewing the code for the operator logic, the CRD definitions, and the interactions with the Azure API to ensure that they are correctly implemented and follow best practices.

## Review Checklist
- check the namespace polcy-operator is created in the cluster and has the correct permissions to manage Azure Policy resources.
- create the namespace if does not exist and ensure that it is correctly configured for the operator to function.
- check the CRD is installed correctly in the policy-operator namespace and has the correct structure as defined in the policy-definition and policy-assignment skills.
- execute make manifest to generate the CRD manifests and ensure that they are correctly generated based on the defined CRD shapes.
- re-install the crd every time the crd shape is modified to ensure that the changes are applied and reflected in the cluster, use make install to install the CRD in the cluster.
- run the controller in local mode using the make run command and ensure that it can successfully connect to the cluster and manage Azure Policy resources as expected.
- Ensure that the operator logic correctly handles the lifecycle of Azure Policy Definitions and Assignments, including creation, updates, and deletion.