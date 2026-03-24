## Project Overview

This repository contains a **Kubernetes Operator built with Kubebuilder** that manages **Azure Policy resources** through Kubernetes Custom Resources.

The operator allows platform teams to define Azure governance policies using Kubernetes manifests, and the controller translates those resources into Azure Policy API calls.

The operator manages the following Azure resources:

- Azure Policy Definitions
- Azure Policy Initiatives (Policy Set Definitions)
- Azure Policy Assignments
- Azure Policy Exemptions
- Azure Policy Remediations

This project follows the **Kubernetes Operator pattern** and integrates with Azure through the **Azure Resource Manager (ARM) Policy REST API**.

the supportted api kinds are 

- AzurePolicyDefinition
- AzurePolicyInitiative
- AzurePolicyAssignment
- AzurePolicyExemption
- AzurePolicyRemediation