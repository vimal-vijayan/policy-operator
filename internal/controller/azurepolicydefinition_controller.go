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

package controller

import (
	"context"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
)

const azurePolicyDefinitionFinalizer = "governance.platform.io/azurepolicydefinition-finalizer"

// DefinitionService is the interface for managing Azure Policy Definitions.
type DefinitionService interface {
	CreateOrUpdate(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (string, error)
	Delete(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) error
}

// AzurePolicyDefinitionReconciler reconciles a AzurePolicyDefinition object
type AzurePolicyDefinitionReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Service DefinitionService
}

// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicydefinitions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicydefinitions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicydefinitions/finalizers,verbs=update

func (r *AzurePolicyDefinitionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	policyDef := &governancev1alpha1.AzurePolicyDefinition{}
	if err := r.Get(ctx, req.NamespacedName, policyDef); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !policyDef.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.handleDeletion(ctx, policyDef)
	}

	if !controllerutil.ContainsFinalizer(policyDef, azurePolicyDefinitionFinalizer) {
		controllerutil.AddFinalizer(policyDef, azurePolicyDefinitionFinalizer)
		return ctrl.Result{}, r.Update(ctx, policyDef)
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, r.reconcileDefinition(ctx, policyDef)
}

func (r *AzurePolicyDefinitionReconciler) handleDeletion(ctx context.Context, policyDef *governancev1alpha1.AzurePolicyDefinition) error {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(policyDef, azurePolicyDefinitionFinalizer) {
		return nil
	}

	logger.Info("Running finalizer cleanup", "name", policyDef.Name)

	if policyDef.Status.PolicyDefinitionID != "" {
		if err := r.Service.Delete(ctx, policyDef); err != nil {
			r.setCondition(policyDef, metav1.ConditionFalse, "DeleteFailed", err.Error())
			if statusErr := r.Status().Update(ctx, policyDef); statusErr != nil {
				logger.Error(statusErr, FailedStatusError)
			}
			return err
		}
	}

	controllerutil.RemoveFinalizer(policyDef, azurePolicyDefinitionFinalizer)
	return r.Update(ctx, policyDef)
}

func (r *AzurePolicyDefinitionReconciler) reconcileDefinition(ctx context.Context, policyDef *governancev1alpha1.AzurePolicyDefinition) error {
	logger := log.FromContext(ctx)

	policyDefinitionID, err := r.Service.CreateOrUpdate(ctx, policyDef)
	if err != nil {
		logger.Error(err, "failed to create/update policy definition")
		r.setCondition(policyDef, metav1.ConditionFalse, "ReconcileFailed", err.Error())
		if statusErr := r.Status().Update(ctx, policyDef); statusErr != nil {
			logger.Error(statusErr, FailedStatusError)
		}
		return err
	}

	policyDef.Status.PolicyDefinitionID = policyDefinitionID
	policyDef.Status.AppliedVersion = policyDef.Spec.Version
	r.setCondition(policyDef, metav1.ConditionTrue, "Reconciled", "Policy definition successfully reconciled")
	return r.Status().Update(ctx, policyDef)
}

func (r *AzurePolicyDefinitionReconciler) setCondition(def *governancev1alpha1.AzurePolicyDefinition, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&def.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: def.Generation,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AzurePolicyDefinitionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&governancev1alpha1.AzurePolicyDefinition{}).
		Complete(r)
}
