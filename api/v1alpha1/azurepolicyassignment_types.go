/*
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
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// AzurePolicyAssignmentSpec defines the desired state of AzurePolicyAssignment
// +kubebuilder:validation:XValidation:rule="(has(self.policyDefinitionRef) && self.policyDefinitionRef != \"\") != (has(self.policyDefinitionId) && self.policyDefinitionId != \"\")",message="Exactly one of policyDefinitionRef or policyDefinitionId must be specified."
type AzurePolicyAssignmentSpec struct {
	// DisplayName is the display name of the policy assignment.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Description is a human-readable description of the policy assignment.
	// +optional
	Description string `json:"description,omitempty"`

	// PolicyDefinitionRef is the name of an AzurePolicyDefinition CR in the same namespace.
	// The operator resolves this reference and uses the policyDefinitionId from its status.
	// Mutually exclusive with policyDefinitionId.
	// +optional
	PolicyDefinitionRef string `json:"policyDefinitionRef,omitempty"`

	// PolicyDefinitionID is the Azure resource ID of the policy definition or initiative to assign.
	// Mutually exclusive with policyDefinitionRef.
	// +optional
	PolicyDefinitionID string `json:"policyDefinitionId,omitempty"`

	// Scope is the Azure resource scope at which the assignment applies.
	// Examples: /subscriptions/{subId}, /subscriptions/{subId}/resourceGroups/{rg},
	// /providers/Microsoft.Management/managementGroups/{mgId}
	// +kubebuilder:validation:Required
	Scope string `json:"scope"`

	// NotScopes is a list of resource scopes excluded from the assignment.
	// +optional
	NotScopes []string `json:"notScopes,omitempty"`

	// EnforcementMode controls whether the policy is enforced.
	// +kubebuilder:validation:Enum=Default;DoNotEnforce
	// +kubebuilder:default=Default
	EnforcementMode string `json:"enforcementMode,omitempty"`

	// Parameters are the parameter values for the assigned policy definition or initiative.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`

	// Metadata is additional metadata for the assignment as a raw JSON object.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *runtime.RawExtension `json:"metadata,omitempty"`

	// Identity configures the managed identity associated with the assignment.
	// Required for policies with deployIfNotExists or modify effects.
	// +optional
	Identity *AssignmentIdentity `json:"identity,omitempty"`

	// NonComplianceMessage is the message to display when the policy assignment is not compliant.
	// +optional
	NonComplianceMessage string `json:"nonComplianceMessage,omitempty"`

	// Exemptions is an optional list of inline exemptions to create for this assignment.
	// +optional
	Exemptions []AssignmentExemptionSpec `json:"exemptions,omitempty"`
}

// AssignmentExemptionSpec defines an inline exemption scoped to a policy assignment.
type AssignmentExemptionSpec struct {
	// DisplayName is the display name of the exemption.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Description is a human-readable description of the exemption.
	// +optional
	Description string `json:"description,omitempty"`

	// Scope is the Azure resource scope at which the exemption applies.
	// +kubebuilder:validation:Required
	Scope string `json:"scope"`

	// ExemptionCategory is the category of the exemption.
	// +kubebuilder:validation:Enum=Waiver;Mitigated
	// +kubebuilder:default=Waiver
	ExemptionCategory string `json:"exemptionCategory,omitempty"`

	// ExpiresOn is the expiration date and time (UTC ISO 8601) of the exemption.
	// +optional
	ExpiresOn string `json:"expiresOn,omitempty"`
}

// AssignmentIdentity defines the managed identity for a policy assignment.
type AssignmentIdentity struct {
	// Type is the identity type.
	// +kubebuilder:validation:Enum=SystemAssigned;UserAssigned;None
	// +kubebuilder:default=None
	Type string `json:"type"`

	// UserAssignedIdentityID is the resource ID of the user-assigned managed identity.
	// Required when Type is UserAssigned.
	// +optional
	UserAssignedIdentityID string `json:"userAssignedIdentityId,omitempty"`

	// Required for SystemAssigned or UserAssigned identity types. The Azure region where the managed identity is created. This is needed to ensure the identity is created in the same region as the policy assignment, as required by Azure for policy assignments with deployIfNotExists or modify effects. If not specified, it defaults to "westeurope".
	// Location is the Azure region where the managed identity is created.
	// +kubebuilder:validation:Enum=eastus;westus;westus2;eastus2;northeurope;westeurope;southeastasia;eastasia;australiaeast;australiasoutheast;brazilsouth;canadacentral;canadaeast;centralindia;southindia;westindia;japaneast;japanwest;koreacentral;koreasouth;southafricanorth;uaenorth;uksouth;ukwest;centralus;southcentralus;northcentralus;westcentralus
	// +kubebuilder:default=westeurope
	Location string `json:"location,omitempty"`
}

// AssignmentExemptionStatus tracks an inline exemption created in Azure for a policy assignment.
type AssignmentExemptionStatus struct {
	// DisplayName matches the exemption spec entry.
	DisplayName string `json:"displayName"`

	// ExemptionID is the Azure resource ID of the created exemption.
	ExemptionID string `json:"exemptionId"`

	// Scope is the Azure resource scope of the exemption, stored for deletion.
	Scope string `json:"scope"`
}

// AzurePolicyAssignmentStatus defines the observed state of AzurePolicyAssignment
type AzurePolicyAssignmentStatus struct {
	// AssignmentID is the Azure resource ID of the created policy assignment.
	// +optional
	AssignmentID string `json:"assignmentId,omitempty"`

	// AssignedLocation is the Azure location set on the policy assignment (required when using managed identity).
	// Persisted so it can be preserved on updates even if identity is removed from the spec.
	// +optional
	AssignedLocation string `json:"assignedLocation,omitempty"`

	// MIPrincipalID is the principal ID of the managed identity associated with the assignment, if any.
	// +optional
	MIPrincipalID string `json:"miPrincipalId,omitempty"`

	// Exemptions tracks the Azure resource IDs of inline exemptions created for this assignment.
	// +optional
	Exemptions []AssignmentExemptionStatus `json:"exemptions,omitempty"`

	// Conditions represent the latest available observations of the resource state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.scope`
// +kubebuilder:printcolumn:name="EnforcementMode",type=string,JSONPath=`.spec.enforcementMode`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AzurePolicyAssignment is the Schema for the azurepolicyassignments API
type AzurePolicyAssignment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzurePolicyAssignmentSpec   `json:"spec,omitempty"`
	Status AzurePolicyAssignmentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzurePolicyAssignmentList contains a list of AzurePolicyAssignment
type AzurePolicyAssignmentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzurePolicyAssignment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzurePolicyAssignment{}, &AzurePolicyAssignmentList{})
}
