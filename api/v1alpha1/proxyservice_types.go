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

// ProxyServiceSpec defines the desired state of ProxyService
type ProxyServiceSpec struct {
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=10
	Replicas int32 `json:"replicas"`

	// +kubebuilder:validation:Minimum=1024
	// +kubebuilder:validation:Maximum=65535
	ListenPort int32 `json:"listenPort"`

	Upstreams []string `json:"upstreams"`

	// +kubebuilder:validation:Minimum=0
	RateLimit int32 `json:"rateLimit"`
}

// ProxyServiceStatus defines the observed state of ProxyService.
type ProxyServiceStatus struct {
	AvailableReplicas int32 `json:"availableReplicas"`

	ActiveUpstreams []string `json:"activeUpstreams"`

	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the ProxyService resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional

}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ProxyService is the Schema for the proxyservices API
type ProxyService struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of ProxyService
	// +required
	Spec ProxyServiceSpec `json:"spec"`

	// status defines the observed state of ProxyService
	// +optional
	Status ProxyServiceStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ProxyServiceList contains a list of ProxyService
type ProxyServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ProxyService `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &ProxyService{}, &ProxyServiceList{})
		return nil
	})
}
