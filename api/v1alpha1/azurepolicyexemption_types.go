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
)

// AzurePolicyExemptionSpec defines the desired state of AzurePolicyExemption
type AzurePolicyExemptionSpec struct {
	// DisplayName is the display name of the policy exemption.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Description is a human-readable description of the policy exemption.
	// +optional
	Description string `json:"description,omitempty"`

	// PolicyAssignmentID is the Azure resource ID of the policy assignment being exempted.
	// +optional
	PolicyAssignmentID string `json:"policyAssignmentId,omitempty"`

	// PolicyAssignmentRef is a reference to the AzurePolicyAssignment custom resource being exempted. Either PolicyAssignmentID or PolicyAssignmentRef must be specified.
	// the operator will use the reference to look up the policy assignment and populate the PolicyAssignmentID if not already set.
	// mutually exclusive with PolicyAssignmentID
	// +optional
	PolicyAssignmentRef string `json:"policyAssignmentRef,omitempty"`

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

	// ResourceSelectors selects resources within the scope to apply the exemption to.
	// +optional
	ResourceSelectors []ResourceSelectorSpec `json:"resourceSelectors,omitempty"`
}

// AzurePolicyExemptionStatus defines the observed state of AzurePolicyExemption
type AzurePolicyExemptionStatus struct {
	// ExemptionID is the Azure resource ID of the created policy exemption.
	// +optional
	ExemptionID string `json:"exemptionId,omitempty"`

	// Conditions represent the latest available observations of the resource state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Scope",type=string,JSONPath=`.spec.scope`
// +kubebuilder:printcolumn:name="Category",type=string,JSONPath=`.spec.exemptionCategory`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// AzurePolicyExemption is the Schema for the azurepolicyexemptions API
type AzurePolicyExemption struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzurePolicyExemptionSpec   `json:"spec,omitempty"`
	Status AzurePolicyExemptionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzurePolicyExemptionList contains a list of AzurePolicyExemption
type AzurePolicyExemptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzurePolicyExemption `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzurePolicyExemption{}, &AzurePolicyExemptionList{})
}
