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
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
	"github.com/vimal-vijayan/azure-policy-operator/internal/service/policyassignment"
)

const azurePolicyAssignmentFinalizer = "governance.platform.io/azurepolicyassignment-finalizer"

// AzurePolicyAssignmentReconciler reconciles a AzurePolicyAssignment object
type AzurePolicyAssignmentReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Service *policyassignment.Service
}

// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyassignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyassignments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyassignments/finalizers,verbs=update

func (r *AzurePolicyAssignmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	assignment := &governancev1alpha1.AzurePolicyAssignment{}
	if err := r.Get(ctx, req.NamespacedName, assignment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !assignment.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(assignment, azurePolicyAssignmentFinalizer) {
			logger.Info("Running finalizer cleanup", "name", assignment.Name)

			if assignment.Status.AssignmentID != "" {
				if err := r.Service.Delete(ctx, assignment.Spec.Scope, assignment.Status.AssignmentID); err != nil {
					r.setCondition(assignment, "Ready", metav1.ConditionFalse, "DeleteFailed", err.Error())
					if statusErr := r.Status().Update(ctx, assignment); statusErr != nil {
						logger.Error(statusErr, "failed to update status")
					}
					return ctrl.Result{}, err
				}
			}

			controllerutil.RemoveFinalizer(assignment, azurePolicyAssignmentFinalizer)
			if err := r.Update(ctx, assignment); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(assignment, azurePolicyAssignmentFinalizer) {
		controllerutil.AddFinalizer(assignment, azurePolicyAssignmentFinalizer)
		if err := r.Update(ctx, assignment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Create or update the Azure Policy Assignment
	assignmentID, err := r.Service.CreateOrUpdate(ctx, assignment)
	if err != nil {
		logger.Error(err, "failed to create/update policy assignment")
		r.setCondition(assignment, "Ready", metav1.ConditionFalse, "ReconcileFailed", err.Error())
		if statusErr := r.Status().Update(ctx, assignment); statusErr != nil {
			logger.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{}, err
	}

	assignment.Status.AssignmentID = assignmentID
	r.setCondition(assignment, "Ready", metav1.ConditionTrue, "Reconciled", "Policy assignment successfully reconciled")
	if err := r.Status().Update(ctx, assignment); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *AzurePolicyAssignmentReconciler) setCondition(assignment *governancev1alpha1.AzurePolicyAssignment, condType string, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&assignment.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: assignment.Generation,
	})
}

// SetupWithManager sets up the controller with the Manager.
func (r *AzurePolicyAssignmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&governancev1alpha1.AzurePolicyAssignment{}).
		Complete(r)
}
