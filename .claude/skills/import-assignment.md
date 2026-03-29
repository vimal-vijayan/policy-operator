---
name: import-assignment
description: This skill explains how to design and implement import/adoption for existing Azure Policy Assignments in a Kubernetes-based Azure Policy Operator.
--- 


# Purpose

It focuses especially on:
	•	annotation-based import
	•	import modes
	•	safe adoption of existing Azure Policy Assignments
	•	controller decision flow
	•	status and condition design

⸻

## Core Design Principle

For import, keep these concerns separate:
	•	annotations = operator instructions
	•	spec = desired state
	•	status = observed state and long-term binding

Recommended binding model:
	•	metadata.annotations["governance.platform.io/import-id"] = one-time adoption hint
	•	metadata.annotations["governance.platform.io/import-mode"] = behavior during/after adoption
	•	status.id = permanent Azure resource binding after import

After successful import, the operator should rely on status.id, not the annotation, for future reconciliation.

⸻

## Recommended Import Annotations

Use these annotations on the CR:

metadata:
  annotations:
    governance.platform.io/import-id: "/subscriptions/<subId>/providers/Microsoft.Authorization/policyAssignments/<assignmentName>"
    governance.platform.io/import-mode: "observe-only"

Meaning
	•	import-id: Full Azure resource ID of the existing Policy Assignment
	•	import-mode: Controls what the operator does after adopting the resource

⸻

## Import Modes

1. observe-only

Use this mode to safely adopt an existing Azure Policy Assignment without modifying Azure.

Behavior:
	•	find the remote assignment using import-id
	•	confirm the resource is a Policy Assignment
	•	record the Azure ID into status.id
	•	compare live Azure state with spec
	•	report drift in conditions
	•	do not update Azure

This is best for:
	•	first-time onboarding
	•	migration from portal/manual/EPAC-managed resources
	•	safe GitOps adoption

2. reconcile

Use this mode when the CR should become authoritative.

Behavior:
	•	adopt the existing assignment if needed
	•	compare live Azure state with spec
	•	update Azure so that it matches the CR

This is best for:
	•	full operator ownership
	•	GitOps-managed governance
	•	post-validation enforcement

3. once

Use this mode for one-time adoption.

Behavior:
	•	import the remote assignment
	•	set status.id
	•	mark import successful
	•	future reconciliation uses status.id
	•	annotation can be ignored or removed afterward

This is best for:
	•	simple one-time migration
	•	workflows where import behavior should not remain active

⸻

Recommended Default

Default to:

import-mode: observe-only

Reason:
	•	safest option
	•	prevents accidental overwrite of production assignments
	•	lets users validate the YAML before enforcement

⸻

## Policy Assignment Import Example

Example CR in observe-only mode

apiVersion: governance.platform.io/v1alpha1
kind: AzurePolicyAssignment
metadata:
  name: audit-vms-dr
  annotations:
    governance.platform.io/import-id: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/Microsoft.Authorization/policyAssignments/audit-vms-without-disaster-recovery"
    governance.platform.io/import-mode: "observe-only"
spec:
  displayName: "Operator Sample - Audit virtual machines without disaster recovery configured"
  description: "Checks whether disaster recovery is configured for virtual machines."
  policyDefinitionId: "/providers/Microsoft.Authorization/policyDefinitions/0015ea4d-51ff-4ce3-8d8c-f3f8f0179a56"
  scope: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018"
  notScopes:
    - "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/resourceGroups/rg-ignore-dr"
  enforcementMode: "Default"
  identity:
    type: "SystemAssigned"
    location: "westeurope"


⸻

What observe-only means in practice

observe-only does not generate YAML automatically.

Instead, it means:
	1.	Attach: Bind the CR to an existing Azure Policy Assignment
	2.	Inspect: Read the actual remote configuration from Azure
	3.	Compare: Detect differences between the CR spec and Azure
	4.	Do not modify: Leave Azure untouched

This mode is effectively a safe validation and onboarding phase.

It is useful for “recording” existing policy assignments into YAML in the sense that the YAML becomes the place where you describe the resource, while the operator helps you detect whether that YAML accurately matches the live Azure object.

⸻

## Recommended Reconcile Precedence

For Policy Assignments, use this decision order:

1. status.id
2. metadata.annotations[import-id]
3. create new resource

Meaning
	•	If status.id exists, the operator already knows the Azure resource to manage
	•	If status.id does not exist but import-id is present, perform adoption/import
	•	If neither exists, create a new Policy Assignment

This avoids repeated dependency on annotations after import.

⸻

## Recommended Controller Behavior for Policy Assignment Import

Case 1: Existing binding in status.id

Behavior:
	•	fetch the Policy Assignment using status.id
	•	compare Azure state with spec
	•	act according to the selected mode or normal reconciliation behavior

Case 2: No status.id, but import-id exists

Behavior:
	•	fetch the remote Azure object by import-id
	•	validate that it is a Policy Assignment
	•	validate it is appropriate for this CR kind
	•	set status.id
	•	set Imported=True
	•	handle post-import behavior based on import-mode

Case 3: No status.id and no import-id

Behavior:
	•	create a new Policy Assignment in Azure
	•	record the created ID in status.id

⸻

## Post-Import Behavior by Mode

observe-only

After import:
	•	compare fields such as:
	•	displayName
	•	description
	•	policyDefinitionId
	•	scope
	•	notScopes
	•	enforcementMode
	•	identity
	•	surface differences as status conditions
	•	do not call update on Azure

reconcile

After import:
	•	compare live state with spec
	•	update Azure Assignment where allowed and needed
	•	clear drift when Azure matches the CR

once

After import:
	•	set status.id
	•	mark successful import
	•	ignore or remove the import annotation afterward
	•	continue normal reconciliation using status.id

⸻

Example Status After Successful Import in observe-only

status:
  id: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/Microsoft.Authorization/policyAssignments/audit-vms-without-disaster-recovery"
  observedGeneration: 1
  conditions:
    - type: Imported
      status: "True"
      reason: ImportSucceeded
      message: "Existing Azure Policy Assignment was adopted successfully."
    - type: DriftDetected
      status: "True"
      reason: SpecMismatch
      message: "Live Azure assignment differs from desired spec.notScopes and description."
    - type: Ready
      status: "True"
      reason: ObservedOnly
      message: "Resource imported in observe-only mode. No changes applied to Azure."


⸻

Example Status After Successful Import in reconcile

status:
  id: "/subscriptions/f2024049-e6cb-4489-9270-6d0d6cd65018/providers/Microsoft.Authorization/policyAssignments/audit-vms-without-disaster-recovery"
  observedGeneration: 2
  conditions:
    - type: Imported
      status: "True"
      reason: ImportSucceeded
      message: "Existing Azure Policy Assignment was adopted successfully."
    - type: DriftDetected
      status: "False"
      reason: InSync
      message: "Azure assignment matches desired spec."
    - type: Ready
      status: "True"
      reason: Reconciled
      message: "Azure Policy Assignment is reconciled successfully."


⸻

Safety Rules

1. Validate resource type

If the CR kind is AzurePolicyAssignment, the import target must be a Policy Assignment.

If the user points to a Policy Definition or another resource type, reject it.

Example condition:

conditions:
  - type: Imported
    status: "False"
    reason: InvalidImportTarget
    message: "Import ID does not reference a Policy Assignment."

2. Prevent rebinding

If status.id already exists and the annotation later points to a different Azure ID, do not silently switch.

Set a conflict condition instead.

Example:

conditions:
  - type: Ready
    status: "False"
    reason: ImportConflict
    message: "annotation import-id differs from already bound status.id"

3. Keep import out of spec

Do not place importId inside spec unless you have a strong reason.

Why:
	•	import is an operator instruction
	•	import is not part of the desired Azure Policy Assignment itself
	•	annotations are the cleaner place for adoption/bootstrap metadata

⸻

Recommended Workflow for Existing Policy Assignments

Step 1: Create CR with observe-only

metadata:
  annotations:
    governance.platform.io/import-id: "<existing-policy-assignment-id>"
    governance.platform.io/import-mode: "observe-only"

Step 2: Let the operator adopt and inspect

The operator will:
	•	bind the CR to the Azure assignment
	•	read its live configuration
	•	report drift in conditions

Step 3: Fix the YAML until drift is gone

Adjust spec until conditions show the resource is in sync.

Step 4: Switch to reconcile

Once the YAML accurately represents the assignment, switch the annotation to:

governance.platform.io/import-mode: "reconcile"

Now the CR becomes the authoritative source of truth.
