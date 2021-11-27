/*
Copyright 2021.

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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AquaScannerAccountSpec defines the desired state of AquaScannerAccount
type AquaScannerAccountSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of AquaScannerAccount. Edit aquascanneraccount_types.go to remove/update
}

// defines a more finely grained desired state for the CR when interacting with aqua api
// values of these properties should be like "Created" "Not Created"
type AquaScannerAccountAquaObjectState struct {
	ApplicationScope string `json:"applicationScope"`
	PermissionSet    string `json:"permissionSet"`
	User             string `json:"user"`
	Role             string `json:"role"`
}

// AquaScannerAccountStatus defines the observed state of AquaScannerAccount
type AquaScannerAccountStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	CurrentState     AquaScannerAccountAquaObjectState `json:"currentState"`
	State            string                            `json:"Status"`
	AccountName      string                            `json:"accountName"`
	AccountSecret    string                            `json:"accountSecret"`
	metav1.Timestamp `json:"timestamp"`
	Message          string                            `json:"message"`
	DesiredState     AquaScannerAccountAquaObjectState `json:"desiredState"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:shortName=asa
// AquaScannerAccount is the Schema for the aquascanneraccounts API
type AquaScannerAccount struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AquaScannerAccountSpec   `json:"spec,omitempty"`
	Status AquaScannerAccountStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// AquaScannerAccountList contains a list of AquaScannerAccount
type AquaScannerAccountList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AquaScannerAccount `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AquaScannerAccount{}, &AquaScannerAccountList{})
}
