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
	"fmt"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
)

const azurePolicyExemptionFinalizer = "governance.platform.io/azurepolicyexemption-finalizer"

// ExemptionService is the interface for managing Azure Policy Exemptions.
type ExemptionService interface {
	CreateOrUpdate(ctx context.Context, exemption *governancev1alpha1.AzurePolicyExemption) (string, error)
	Delete(ctx context.Context, scope, exemptionID string) error
}

// AzurePolicyExemptionReconciler reconciles a AzurePolicyExemption object
type AzurePolicyExemptionReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Service ExemptionService
}

// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyexemptions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyexemptions/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyexemptions/finalizers,verbs=update

func (r *AzurePolicyExemptionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	exemption := &governancev1alpha1.AzurePolicyExemption{}
	if err := r.Get(ctx, req.NamespacedName, exemption); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !exemption.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(exemption, azurePolicyExemptionFinalizer) {
			logger.Info("Running finalizer cleanup", "name", exemption.Name)

			if exemption.Status.ExemptionID != "" {
				if err := r.Service.Delete(ctx, exemption.Spec.Scope, exemption.Status.ExemptionID); err != nil {
					r.setCondition(exemption, metav1.ConditionFalse, "DeleteFailed", err.Error())
					if statusErr := r.Status().Update(ctx, exemption); statusErr != nil {
						logger.Error(statusErr, FailedStatusError)
					}
					return ctrl.Result{}, err
				}
			}

			controllerutil.RemoveFinalizer(exemption, azurePolicyExemptionFinalizer)
			if err := r.Update(ctx, exemption); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(exemption, azurePolicyExemptionFinalizer) {
		controllerutil.AddFinalizer(exemption, azurePolicyExemptionFinalizer)
		if err := r.Update(ctx, exemption); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Resolve policyAssignmentRef to policyAssignmentId if not already set
	if exemption.Spec.PolicyAssignmentRef != "" && exemption.Spec.PolicyAssignmentID == "" {
		assignment := &governancev1alpha1.AzurePolicyAssignment{}
		if err := r.Get(ctx, types.NamespacedName{Name: exemption.Spec.PolicyAssignmentRef, Namespace: exemption.Namespace}, assignment); err != nil {
			logger.Error(err, "failed to resolve policyAssignmentRef", "ref", exemption.Spec.PolicyAssignmentRef)
			r.setCondition(exemption, metav1.ConditionFalse, "RefResolutionFailed", fmt.Sprintf("policyAssignmentRef %q not found: %v", exemption.Spec.PolicyAssignmentRef, err))
			if statusErr := r.Status().Update(ctx, exemption); statusErr != nil {
				logger.Error(statusErr, FailedStatusError)
			}
			return ctrl.Result{}, err
		}
		if assignment.Status.AssignmentID == "" {
			msg := fmt.Sprintf("policyAssignmentRef %q has not yet been reconciled (AssignmentID is empty)", exemption.Spec.PolicyAssignmentRef)
			logger.Info(msg)
			r.setCondition(exemption, metav1.ConditionFalse, "RefNotReady", msg)
			if statusErr := r.Status().Update(ctx, exemption); statusErr != nil {
				logger.Error(statusErr, FailedStatusError)
			}
			return ctrl.Result{Requeue: true}, nil
		}
		exemption.Spec.PolicyAssignmentID = assignment.Status.AssignmentID
	}

	// Create or update the Azure Policy Exemption
	exemptionID, err := r.Service.CreateOrUpdate(ctx, exemption)
	if err != nil {
		logger.Error(err, "failed to create/update policy exemption")
		r.setCondition(exemption, metav1.ConditionFalse, "ReconcileFailed", err.Error())
		if statusErr := r.Status().Update(ctx, exemption); statusErr != nil {
			logger.Error(statusErr, FailedStatusError)
		}
		return ctrl.Result{}, err
	}

	exemption.Status.ExemptionID = exemptionID
	r.setCondition(exemption, metav1.ConditionTrue, "Reconciled", "Policy exemption successfully reconciled")
	if err := r.Status().Update(ctx, exemption); err != nil {
		logger.Error(err, FailedStatusError)
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, nil
}

func (r *AzurePolicyExemptionReconciler) setCondition(exemption *governancev1alpha1.AzurePolicyExemption, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&exemption.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: exemption.Generation,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AzurePolicyExemptionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&governancev1alpha1.AzurePolicyExemption{}).
		Complete(r)
}
