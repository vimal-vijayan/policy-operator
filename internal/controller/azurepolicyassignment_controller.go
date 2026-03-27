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
	"time"

	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicydefinitions,verbs=get;list;watch

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
				if err := r.Service.Delete(ctx, assignment.Spec.Scope, assignment.Status.AssignmentID, assignment.Status.Exemptions, assignment.Spec.Identity); err != nil {
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

	// Resolve policyDefinitionId — either directly from spec or via policyDefinitionRef
	policyDefinitionID := assignment.Spec.PolicyDefinitionID
	if assignment.Spec.PolicyDefinitionRef != "" {
		policyDef := &governancev1alpha1.AzurePolicyDefinition{}
		if err := r.Get(ctx, types.NamespacedName{Name: assignment.Spec.PolicyDefinitionRef, Namespace: req.Namespace}, policyDef); err != nil {
			r.setCondition(assignment, "Ready", metav1.ConditionFalse, "RefNotFound", fmt.Sprintf("AzurePolicyDefinition %q not found: %v", assignment.Spec.PolicyDefinitionRef, err))
			_ = r.Status().Update(ctx, assignment)
			return ctrl.Result{}, err
		}
		if policyDef.Status.PolicyDefinitionID == "" {
			r.setCondition(assignment, "Ready", metav1.ConditionFalse, "RefNotReady", fmt.Sprintf("AzurePolicyDefinition %q has no policyDefinitionId in status yet", assignment.Spec.PolicyDefinitionRef))
			_ = r.Status().Update(ctx, assignment)
			return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
		}
		policyDefinitionID = policyDef.Status.PolicyDefinitionID
	}

	// Create or update the Azure Policy Assignment (and its inline exemptions and role assignments)
	assignmentID, assignedLocation, miPrincipalID, exemptionStatuses, err := r.Service.CreateOrUpdate(ctx, assignment, policyDefinitionID)
	if assignmentID != "" {
		assignment.Status.AssignmentID = assignmentID
	}
	if assignedLocation != "" {
		assignment.Status.AssignedLocation = assignedLocation
	}
	if miPrincipalID != "" {
		assignment.Status.MIPrincipalID = miPrincipalID
	}
	if exemptionStatuses != nil {
		assignment.Status.Exemptions = exemptionStatuses
	}

	if err != nil {
		logger.Error(err, "failed to create/update policy assignment")
		r.setCondition(assignment, "Ready", metav1.ConditionFalse, "ReconcileFailed", err.Error())
		if statusErr := r.Status().Update(ctx, assignment); statusErr != nil {
			logger.Error(statusErr, "failed to update status")
		}
		return ctrl.Result{}, err
	}

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
		Watches(
			&governancev1alpha1.AzurePolicyDefinition{},
			handler.EnqueueRequestsFromMapFunc(r.assignmentsReferencingDefinition),
		).
		Complete(r)
}

// assignmentsReferencingDefinition maps an AzurePolicyDefinition event to all
// AzurePolicyAssignments that reference it via policyDefinitionRef.
func (r *AzurePolicyAssignmentReconciler) assignmentsReferencingDefinition(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &governancev1alpha1.AzurePolicyAssignmentList{}
	if err := r.List(ctx, list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for _, a := range list.Items {
		if a.Spec.PolicyDefinitionRef == obj.GetName() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: a.Name, Namespace: a.Namespace},
			})
		}
	}
	return requests
}
