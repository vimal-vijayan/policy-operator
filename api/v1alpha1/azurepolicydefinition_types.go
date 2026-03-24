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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AzurePolicyDefinitionSpec defines the desired state of AzurePolicyDefinition
// +kubebuilder:validation:XValidation:rule="has(self.policyRule) != has(self.policyRuleJson)",message="Exactly one of policyRule or policyRuleJson must be specified."
type AzurePolicyDefinitionSpec struct {
	// DisplayName is the display name of the policy definition.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Description is a human-readable description of the policy definition.
	// +optional
	Description string `json:"description,omitempty"`

	// PolicyType is the type of policy definition. Allowed values: BuiltIn, Custom, NotSpecified, Static.
	// +kubebuilder:validation:Enum=BuiltIn;Custom;NotSpecified;Static
	// +kubebuilder:default=Custom
	PolicyType string `json:"policyType,omitempty"`

	// Mode determines which resource types are evaluated. Common values: All, Indexed.
	// +kubebuilder:validation:Required
	Mode string `json:"mode"`

	// Version is the semantic version of the policy definition (e.g. "1.0.0").
	// When set, it is injected into the Azure Policy metadata object under the
	// key "version", which is displayed by the Azure portal. spec.version always
	// takes precedence over any "version" key already present in spec.metadata.
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+$`
	Version string `json:"version,omitempty"`

	// Metadata is additional metadata for the policy definition as a raw JSON object.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *runtime.RawExtension `json:"metadata,omitempty"`

	// Parameters defines the parameters that can be used in the policy rule.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`

	// PolicyRule is the logic of the policy definition as a raw JSON object.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	PolicyRule *runtime.RawExtension `json:"policyRule,omitempty"`

	// PolicyRuleJSON is the logic of the policy definition as a JSON string.
	// This is useful when managing very large policy definitions as raw JSON text.
	// +optional
	PolicyRuleJSON string `json:"policyRuleJson,omitempty"`

	// SubscriptionID is the Azure subscription to deploy the policy definition to.
	// If omitted, the policy is created at the management group scope.
	// +optional
	SubscriptionID string `json:"subscriptionId,omitempty"`

	// ManagementGroupID is the management group scope to deploy the policy definition to.
	// +optional
	ManagementGroupID string `json:"managementGroupId,omitempty"`
}

// AzurePolicyDefinitionStatus defines the observed state of AzurePolicyDefinition
type AzurePolicyDefinitionStatus struct {
	// PolicyDefinitionID is the Azure resource ID of the created policy definition.
	// +optional
	PolicyDefinitionID string `json:"policyDefinitionId,omitempty"`

	// AppliedVersion is the semver version last successfully written to Azure
	// Policy metadata. Mirrors spec.version after each successful reconcile.
	// +optional
	AppliedVersion string `json:"appliedVersion,omitempty"`

	// Conditions represent the latest available observations of the resource state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AzurePolicyDefinition is the Schema for the azurepolicydefinitions API
type AzurePolicyDefinition struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzurePolicyDefinitionSpec   `json:"spec,omitempty"`
	Status AzurePolicyDefinitionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzurePolicyDefinitionList contains a list of AzurePolicyDefinition
type AzurePolicyDefinitionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzurePolicyDefinition `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzurePolicyDefinition{}, &AzurePolicyDefinitionList{})
}
