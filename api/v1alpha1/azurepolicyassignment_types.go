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
type AzurePolicyAssignmentSpec struct {
	// DisplayName is the display name of the policy assignment.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Description is a human-readable description of the policy assignment.
	// +optional
	Description string `json:"description,omitempty"`

	// PolicyDefinitionID is the Azure resource ID of the policy definition or initiative to assign.
	// +kubebuilder:validation:Required
	PolicyDefinitionID string `json:"policyDefinitionId"`

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
}

// AzurePolicyAssignmentStatus defines the observed state of AzurePolicyAssignment
type AzurePolicyAssignmentStatus struct {
	// AssignmentID is the Azure resource ID of the created policy assignment.
	// +optional
	AssignmentID string `json:"assignmentId,omitempty"`

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
