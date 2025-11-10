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

package controller

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	labelerv1alpha1 "github.com/Arsolitt/k8s-node-labeler/api/v1alpha1"
)

// LabelMappingReconciler согласовывает объект LabelMapping
type LabelMappingReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=labeler.arsolitt.tech,resources=labelmappings,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=labeler.arsolitt.tech,resources=labelmappings/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=labeler.arsolitt.tech,resources=labelmappings/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=nodes,verbs=get;list;watch;update;patch

// Reconcile является частью основного цикла согласования kubernetes, который направлен на
// перемещение текущего состояния кластера ближе к желаемому состоянию.
func (r *LabelMappingReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Получаем LabelMapping
	labelMapping := &labelerv1alpha1.LabelMapping{}
	if err := r.Get(ctx, req.NamespacedName, labelMapping); err != nil {
		if errors.IsNotFound(err) {
			// Объект не найден, возможно был удален после запроса на согласование
			return ctrl.Result{}, nil
		}
		log.Error(err, "Не удалось получить LabelMapping")
		return ctrl.Result{}, err
	}

	// Получаем все ноды
	nodeList := &corev1.NodeList{}
	if err := r.List(ctx, nodeList); err != nil {
		log.Error(err, "Не удалось получить список нод")
		return ctrl.Result{}, err
	}

	matchedNodes := 0
	labeledNodes := 0

	// Обрабатываем каждую ноду
	for i := range nodeList.Items {
		node := &nodeList.Items[i]

		// Проверяем, соответствует ли нода селектору
		if !r.nodeMatchesSelector(node, labelMapping.Spec.NodeSelector) {
			// Если нода не соответствует селектору, удаляем лейблы
			if r.removeLabelsFromNode(ctx, node, labelMapping.Spec.Labels) {
				labeledNodes++
			}
			continue
		}

		// Проверяем, соответствует ли нода всем условиям
		if r.nodeMatchesConditions(node, labelMapping.Spec.Conditions) {
			matchedNodes++
			// Если соответствует, добавляем лейблы
			if r.addLabelsToNode(ctx, node, labelMapping.Spec.Labels) {
				labeledNodes++
			}
		} else {
			// Если не соответствует условиям, удаляем лейблы
			if r.removeLabelsFromNode(ctx, node, labelMapping.Spec.Labels) {
				labeledNodes++
			}
		}
	}

	// Обновляем статус LabelMapping
	now := metav1.Now()
	labelMapping.Status.MatchedNodes = matchedNodes
	labelMapping.Status.LabeledNodes = labeledNodes
	labelMapping.Status.LastSyncTime = &now

	// Устанавливаем условие Ready
	condition := metav1.Condition{
		Type:               "Ready",
		Status:             metav1.ConditionTrue,
		Reason:             "ReconciliationSucceeded",
		Message:            fmt.Sprintf("Successfully reconciled %d nodes", labeledNodes),
		LastTransitionTime: now,
	}

	// Обновляем или добавляем условие
	found := false
	for i, cond := range labelMapping.Status.Conditions {
		if cond.Type == condition.Type {
			labelMapping.Status.Conditions[i] = condition
			found = true
			break
		}
	}
	if !found {
		labelMapping.Status.Conditions = append(labelMapping.Status.Conditions, condition)
	}

	if err := r.Status().Update(ctx, labelMapping); err != nil {
		log.Error(err, "Не удалось обновить статус LabelMapping")
		return ctrl.Result{}, err
	}

	log.Info("Reconciliation completed",
		"matchedNodes", matchedNodes,
		"labeledNodes", labeledNodes)

	// Пересогласовываем каждые 30 секунд для отслеживания изменений
	return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

// nodeMatchesSelector проверяет, соответствует ли нода селектору
func (r *LabelMappingReconciler) nodeMatchesSelector(node *corev1.Node, selector map[string]string) bool {
	if len(selector) == 0 {
		// Если селектор пуст, соответствуют все ноды
		return true
	}

	nodeLabels := labels.Set(node.Labels)
	return labels.SelectorFromSet(selector).Matches(nodeLabels)
}

// nodeMatchesConditions проверяет, соответствует ли нода всем условиям
func (r *LabelMappingReconciler) nodeMatchesConditions(node *corev1.Node, conditions []labelerv1alpha1.NodeCondition) bool {
	for _, condition := range conditions {
		switch condition.Type {
		case labelerv1alpha1.ConditionTypeNodeReady:
			if !r.checkNodeReadyCondition(node, condition.Status) {
				return false
			}
		}
	}
	return true
}

// checkNodeReadyCondition проверяет условие готовности ноды
func (r *LabelMappingReconciler) checkNodeReadyCondition(node *corev1.Node, expectedStatus corev1.ConditionStatus) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == expectedStatus
		}
	}
	return false
}

// addLabelsToNode добавляет лейблы на ноду, возвращает true если были изменения
func (r *LabelMappingReconciler) addLabelsToNode(ctx context.Context, node *corev1.Node, labels map[string]string) bool {
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	hasChanges := false
	for key, value := range labels {
		if currentValue, exists := node.Labels[key]; !exists || currentValue != value {
			node.Labels[key] = value
			hasChanges = true
		}
	}

	if hasChanges {
		if err := r.Update(ctx, node); err != nil {
			log.FromContext(ctx).Error(err, "Не удалось обновить лейблы на ноде", "node", node.Name)
			return false
		}
		log.FromContext(ctx).Info("Добавлены лейблы на ноду", "node", node.Name, "labels", labels)
	}

	return hasChanges
}

// removeLabelsFromNode удаляет лейблы с ноды, возвращает true если были изменения
func (r *LabelMappingReconciler) removeLabelsFromNode(ctx context.Context, node *corev1.Node, labels map[string]string) bool {
	if node.Labels == nil {
		return false
	}

	hasChanges := false
	for key := range labels {
		if _, exists := node.Labels[key]; exists {
			delete(node.Labels, key)
			hasChanges = true
		}
	}

	if hasChanges {
		if err := r.Update(ctx, node); err != nil {
			log.FromContext(ctx).Error(err, "Не удалось удалить лейблы с ноды", "node", node.Name)
			return false
		}
		log.FromContext(ctx).Info("Удалены лейблы с ноды", "node", node.Name, "labels", labels)
	}

	return hasChanges
}

// SetupWithManager настраивает контроллер с Manager
func (r *LabelMappingReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&labelerv1alpha1.LabelMapping{}).
		Watches(
			&corev1.Node{},
			handler.EnqueueRequestsFromMapFunc(r.findLabelMappingsForNode),
		).
		Complete(r)
}

// findLabelMappingsForNode находит все LabelMapping, которые должны обработать изменения ноды
func (r *LabelMappingReconciler) findLabelMappingsForNode(ctx context.Context, obj client.Object) []reconcile.Request {
	// Получаем все LabelMapping
	labelMappingList := &labelerv1alpha1.LabelMappingList{}
	if err := r.List(ctx, labelMappingList); err != nil {
		log.FromContext(ctx).Error(err, "Не удалось получить список LabelMapping")
		return []reconcile.Request{}
	}

	// Создаем запросы на согласование для всех LabelMapping
	requests := make([]reconcile.Request, len(labelMappingList.Items))
	for i, item := range labelMappingList.Items {
		requests[i] = reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name: item.GetName(),
			},
		}
	}

	return requests
}
