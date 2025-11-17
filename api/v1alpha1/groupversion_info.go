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

// Package v1alpha1 содержит API Schema определения для labeler v1alpha1 API группы
// +kubebuilder:object:generate=true
// +groupName=labeler.arsolitt.tech
package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

var (
	// GroupVersion является версией группы, используемой для регистрации этих объектов
	GroupVersion = schema.GroupVersion{Group: "labeler.arsolitt.tech", Version: "v1alpha1"}

	// SchemeBuilder используется для добавления go типов в GroupVersionKind схему
	SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

	// AddToScheme добавляет типы в этой группе-версии в данную схему.
	AddToScheme = SchemeBuilder.AddToScheme
)
