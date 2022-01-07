/*
Copyright 2021 Pragmatics.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SuiteSpec defines the desired state of Suite
type SuiteSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Tests    []TestSpec `json:"tests"`
	Capacity int        `json:"capacity,omitempty"`
}

type TestSpec struct {
	Name                  string `json:"name"`
	TektonPipelineRunYaml string `json:"tektonPipelineRunYaml"`
	MaxRetries            int    `json:"maxRetries,omitempty"`
	RunInIsolation        bool   `json:"runInIsolation,omitempty"`
}

// SuiteStatus defines the observed state of Suite
type SuiteStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Status string       `json:"status,omitempty"`
	Tests  []TestStatus `json:"tests,omitempty"`
}

type TestStatus struct {
	Name     string `json:"name,omitempty"`
	Status   string `json:"status,omitempty"`
	Attempts int    `json:"attempts,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Suite is the Schema for the suites API
type Suite struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SuiteSpec   `json:"spec,omitempty"`
	Status SuiteStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SuiteList contains a list of Suite
type SuiteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Suite `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Suite{}, &SuiteList{})
}
