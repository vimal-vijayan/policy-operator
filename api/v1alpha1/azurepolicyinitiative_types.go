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

// PolicyDefinitionReference specifies a policy definition included in the initiative.
// Exactly one of policyDefinitionId or policyDefinitionRef must be set.
// +kubebuilder:validation:XValidation:rule="has(self.policyDefinitionId) != has(self.policyDefinitionRef)",message="Exactly one of policyDefinitionId or policyDefinitionRef must be specified."
type PolicyDefinitionReference struct {
	// PolicyDefinitionID is the Azure resource ID of a built-in or custom policy definition.
	// +optional
	PolicyDefinitionID string `json:"policyDefinitionId,omitempty"`

	// PolicyDefinitionRef is the name of an AzurePolicyDefinition CR in the same namespace.
	// The operator resolves this reference to the Azure resource ID at reconcile time.
	// +optional
	PolicyDefinitionRef string `json:"policyDefinitionRef,omitempty"`

	// Parameters are parameter values passed to the referenced policy definition.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Parameters *runtime.RawExtension `json:"parameters,omitempty"`
}

// InitiativeParameter defines a parameter accepted by the policy set definition.
type InitiativeParameter struct {
	// Name is the parameter name.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Type is the data type of the parameter.
	// +kubebuilder:validation:Enum=String;Array;Object;Boolean;Integer;Float;DateTime
	Type string `json:"type"`

	// DefaultValue is the default value if no value is supplied at assignment time.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	DefaultValue *runtime.RawExtension `json:"defaultValue,omitempty"`

	// AllowedValues restricts the set of values that can be supplied.
	// +optional
	AllowedValues []string `json:"allowedValues,omitempty"`

	// StrongType provides an Azure portal UI hint (e.g. "location", "resourceTypes").
	// +optional
	StrongType string `json:"strongType,omitempty"`
}

// AzurePolicyInitiativeSpec defines the desired state of AzurePolicyInitiative
type AzurePolicyInitiativeSpec struct {
	// DisplayName is the display name of the policy set definition.
	// +kubebuilder:validation:Required
	DisplayName string `json:"displayName"`

	// Description is a human-readable description of the initiative.
	// +optional
	Description string `json:"description,omitempty"`

	// Version is the semantic version of the initiative (e.g. "1.0.0").
	// When set, it is injected into the Azure Policy metadata under the key "version".
	// +optional
	// +kubebuilder:validation:Pattern=`^\d+\.\d+\.\d+$`
	Version string `json:"version,omitempty"`

	// Metadata is additional metadata for the initiative as a raw JSON object.
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Metadata *runtime.RawExtension `json:"metadata,omitempty"`

	// Parameters defines the parameters accepted by the initiative.
	// +optional
	Parameters []InitiativeParameter `json:"parameters,omitempty"`

	// PolicyDefinitions is the list of policy definitions included in the initiative.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinItems=1
	PolicyDefinitions []PolicyDefinitionReference `json:"policyDefinitions"`

	// SubscriptionID is the Azure subscription to deploy the initiative to.
	// If omitted and managementGroupId is not set, the operator subscription is used.
	// +optional
	SubscriptionID string `json:"subscriptionId,omitempty"`

	// ManagementGroupID is the management group scope to deploy the initiative to.
	// +optional
	ManagementGroupID string `json:"managementGroupId,omitempty"`
}

// AzurePolicyInitiativeStatus defines the observed state of AzurePolicyInitiative
type AzurePolicyInitiativeStatus struct {
	// InitiativeID is the Azure resource ID of the created policy set definition.
	// +optional
	InitiativeID string `json:"initiativeId,omitempty"`

	// AppliedVersion is the semver version last successfully written to Azure.
	// +optional
	AppliedVersion string `json:"appliedVersion,omitempty"`

	// Conditions represent the latest available observations of the resource state.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Ready",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].status",description="Indicates if the initiative is ready"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Time since creation"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason",description="Reason for the current status"
// +kubebuilder:printcolumn:name="ID",type="string",JSONPath=".status.initiativeId",description="Azure resource ID of the initiative"

// AzurePolicyInitiative is the Schema for the azurepolicyinitiatives API
type AzurePolicyInitiative struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AzurePolicyInitiativeSpec   `json:"spec,omitempty"`
	Status AzurePolicyInitiativeStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AzurePolicyInitiativeList contains a list of AzurePolicyInitiative
type AzurePolicyInitiativeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AzurePolicyInitiative `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AzurePolicyInitiative{}, &AzurePolicyInitiativeList{})
}
