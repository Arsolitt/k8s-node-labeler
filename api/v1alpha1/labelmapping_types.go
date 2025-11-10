/*
Copyright 2025.

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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ConditionType определяет тип условия для проверки
type ConditionType string

const (
	// ConditionTypeNodeReady проверяет готовность ноды
	ConditionTypeNodeReady ConditionType = "NodeReady"
)

// NodeCondition определяет условие, которому должна соответствовать нода
type NodeCondition struct {
	// Type определяет тип условия (например, NodeReady)
	// +kubebuilder:validation:Enum=NodeReady
	Type ConditionType `json:"type"`

	// Status определяет ожидаемый статус условия (True/False)
	// +kubebuilder:validation:Enum=True;False
	Status corev1.ConditionStatus `json:"status"`
}

// LabelMappingSpec определяет желаемое состояние LabelMapping
type LabelMappingSpec struct {
	// NodeSelector используется для выбора нод, к которым будет применяться маппинг
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Conditions определяет условия, которым должна соответствовать нода
	// Все условия должны быть выполнены (логика AND)
	// +kubebuilder:validation:MinItems=1
	Conditions []NodeCondition `json:"conditions"`

	// Labels определяет лейблы, которые должны быть добавлены на ноду,
	// если она соответствует всем условиям, и удалены, если не соответствует
	// +kubebuilder:validation:MinProperties=1
	Labels map[string]string `json:"labels"`
}

// LabelMappingStatus определяет наблюдаемое состояние LabelMapping
type LabelMappingStatus struct {
	// MatchedNodes содержит количество нод, соответствующих условиям
	MatchedNodes int `json:"matchedNodes"`

	// LabeledNodes содержит количество нод, на которые были применены лейблы
	LabeledNodes int `json:"labeledNodes"`

	// LastSyncTime содержит время последней синхронизации
	LastSyncTime *metav1.Time `json:"lastSyncTime,omitempty"`

	// Conditions содержит текущие условия состояния ресурса
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:printcolumn:name="Matched",type=integer,JSONPath=`.status.matchedNodes`
// +kubebuilder:printcolumn:name="Labeled",type=integer,JSONPath=`.status.labeledNodes`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// LabelMapping является схемой для API labelmappings
type LabelMapping struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LabelMappingSpec   `json:"spec,omitempty"`
	Status LabelMappingStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LabelMappingList содержит список LabelMapping
type LabelMappingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LabelMapping `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LabelMapping{}, &LabelMappingList{})
}
