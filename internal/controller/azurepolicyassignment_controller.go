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
	"strings"

	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
)

const azurePolicyAssignmentFinalizer = "governance.platform.io/azurepolicyassignment-finalizer"

// PolicyAssignmentService defines the Azure operations required by the assignment controller.
type PolicyAssignmentService interface {
	CreateOrUpdate(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error)
	Delete(ctx context.Context, scope string, assignmentID string, exemptions []governancev1alpha1.AssignmentExemptionStatus, identity *governancev1alpha1.AssignmentIdentity) error
	Import(ctx context.Context, importID string, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, []string, error)
}

// AzurePolicyAssignmentReconciler reconciles a AzurePolicyAssignment object
type AzurePolicyAssignmentReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Service  PolicyAssignmentService
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyassignments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyassignments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyassignments/finalizers,verbs=update
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicydefinitions,verbs=get;list;watch

func (r *AzurePolicyAssignmentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	assignment := &governancev1alpha1.AzurePolicyAssignment{}
	if err := r.Get(ctx, req.NamespacedName, assignment); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !assignment.DeletionTimestamp.IsZero() {
		return r.handleDeletion(ctx, assignment)
	}

	if !controllerutil.ContainsFinalizer(assignment, azurePolicyAssignmentFinalizer) {
		controllerutil.AddFinalizer(assignment, azurePolicyAssignmentFinalizer)
		if err := r.Update(ctx, assignment); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	policyDefinitionID, result, done, err := r.resolvePolicyDefinitionID(ctx, req, assignment)
	if done {
		return result, err
	}

	result, done, err = r.handleImport(ctx, assignment, policyDefinitionID)
	if done {
		return result, err
	}

	return r.reconcileCreateOrUpdate(ctx, assignment, policyDefinitionID)
}

func (r *AzurePolicyAssignmentReconciler) handleDeletion(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(assignment, azurePolicyAssignmentFinalizer) {
		return ctrl.Result{}, nil
	}

	logger.Info("Running finalizer cleanup", "name", assignment.Name)

	if assignment.Status.AssignmentID != "" {
		if err := r.Service.Delete(ctx, assignment.Spec.Scope, assignment.Status.AssignmentID, assignment.Status.Exemptions, assignment.Spec.Identity); err != nil {
			if r.Recorder != nil {
				r.Recorder.Eventf(assignment, corev1.EventTypeWarning, "PolicyAssignmentDeleteFailed", "Failed deleting policy assignment %q: %v", assignment.Status.AssignmentID, err)
			}
			r.setCondition(assignment, metav1.ConditionFalse, "DeleteFailed", err.Error())
			if statusErr := r.Status().Update(ctx, assignment); statusErr != nil {
				logger.Error(statusErr, FailedStatusError)
			}
			return ctrl.Result{}, err
		}
		if r.Recorder != nil {
			r.Recorder.Eventf(assignment, corev1.EventTypeNormal, "PolicyAssignmentDeleted", "Deleted policy assignment %q", assignment.Status.AssignmentID)
		}
	}

	controllerutil.RemoveFinalizer(assignment, azurePolicyAssignmentFinalizer)
	return ctrl.Result{}, r.Update(ctx, assignment)
}

func (r *AzurePolicyAssignmentReconciler) resolvePolicyDefinitionID(ctx context.Context, req ctrl.Request, assignment *governancev1alpha1.AzurePolicyAssignment) (string, ctrl.Result, bool, error) {
	if assignment.Spec.PolicyDefinitionRef == "" {
		return assignment.Spec.PolicyDefinitionID, ctrl.Result{}, false, nil
	}

	policyDef := &governancev1alpha1.AzurePolicyDefinition{}
	if err := r.Get(ctx, types.NamespacedName{Name: assignment.Spec.PolicyDefinitionRef, Namespace: req.Namespace}, policyDef); err != nil {
		r.setCondition(assignment, metav1.ConditionFalse, "RefNotFound", fmt.Sprintf("AzurePolicyDefinition %q not found: %v", assignment.Spec.PolicyDefinitionRef, err))
		_ = r.Status().Update(ctx, assignment)
		return "", ctrl.Result{}, true, err
	}
	if policyDef.Status.PolicyDefinitionID == "" {
		r.setCondition(assignment, metav1.ConditionFalse, "RefNotReady", fmt.Sprintf("AzurePolicyDefinition %q has no policyDefinitionId in status yet", assignment.Spec.PolicyDefinitionRef))
		_ = r.Status().Update(ctx, assignment)
		return "", ctrl.Result{RequeueAfter: DefaultRequeueDuration}, true, nil
	}
	return policyDef.Status.PolicyDefinitionID, ctrl.Result{}, false, nil
}

func (r *AzurePolicyAssignmentReconciler) handleImport(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (ctrl.Result, bool, error) {
	logger := log.FromContext(ctx)
	importID := assignment.Annotations[annotationImportID]

	if importID == "" {
		return ctrl.Result{}, false, nil
	}

	// Prevent rebinding when status already points to a different Azure resource ID.
	// Azure resource IDs are case-insensitive, so compare with EqualFold to avoid
	// spurious conflicts caused by casing differences (e.g. annotation uses lowercase
	// provider namespace while Azure returns canonical casing in the status).
	if assignment.Status.AssignmentID != "" && !strings.EqualFold(importID, assignment.Status.AssignmentID) {
		msg := fmt.Sprintf("annotation import-id %q differs from already bound assignmentId %q", importID, assignment.Status.AssignmentID)
		r.setCondition(assignment, metav1.ConditionFalse, "ImportConflict", msg)
		_ = r.Status().Update(ctx, assignment)
		return ctrl.Result{}, true, fmt.Errorf("import conflict: %s", msg)
	}

	// Already bound — no adoption needed.
	// if assignment.Status.AssignmentID != "" {
	// 	return ctrl.Result{}, false, nil
	// }

	importMode := assignment.Annotations[annotationImportMode]
	if importMode == "" {
		importMode = importModeObserveOnly
	}

	// For "adopt-once" mode: if already applied, skip re-reconciling Azure.
	if importMode == importModeOnlyOnce {
		cond := apimeta.FindStatusCondition(assignment.Status.Conditions, "Ready")
		if cond != nil && cond.Reason == ReasonAppliedOnce {
			logger.V(1).Info("Skipping reconcile — already applied once", "name", assignment.Name)
			return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, true, nil
		}
	}

	logger.Info("Importing existing Azure Policy Assignment", "importID", importID, "importMode", importMode)

	assignedLocation, miPrincipalID, driftFields, err := r.Service.Import(ctx, importID, assignment, policyDefinitionID)
	if err != nil {
		r.setImportedCondition(assignment, metav1.ConditionFalse, "ImportFailed", err.Error())
		r.setCondition(assignment, metav1.ConditionFalse, "ImportFailed", err.Error())
		_ = r.Status().Update(ctx, assignment)
		// This Requeue with backoff to handle transient import errors, but avoid tight looping on unrecoverable issues.
		return ctrl.Result{RequeueAfter: FailedRequeueDuration}, true, err
	}

	assignment.Status.AssignmentID = importID
	if assignedLocation != "" {
		assignment.Status.AssignedLocation = assignedLocation
	}
	if miPrincipalID != "" {
		assignment.Status.MIPrincipalID = miPrincipalID
	}

	r.setImportedCondition(assignment, metav1.ConditionTrue, "ImportSucceeded", "Existing Azure Policy Assignment was adopted successfully.")
	r.setDriftCondition(assignment, driftFields)

	if importMode == importModeObserveOnly {
		r.setCondition(assignment, metav1.ConditionTrue, "ObservedOnly", "Resource imported in observe-only mode. No changes applied to Azure.")
		if err := r.Status().Update(ctx, assignment); err != nil {
			logger.Error(err, FailedStatusError)
			return ctrl.Result{}, true, err
		}
		return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, true, nil
	}

	// For "adopt-once" and "reconcile" modes: fall through to CreateOrUpdate.
	return ctrl.Result{}, false, nil
}

func (r *AzurePolicyAssignmentReconciler) reconcileCreateOrUpdate(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	wasAssigned := assignment.Status.AssignmentID != ""
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
		if r.Recorder != nil {
			r.Recorder.Eventf(assignment, corev1.EventTypeWarning, "PolicyAssignmentReconcileFailed", "Failed creating/updating policy assignment: %v", err)
		}
		r.setCondition(assignment, metav1.ConditionFalse, "ReconcileFailed", err.Error())
		if statusErr := r.Status().Update(ctx, assignment); statusErr != nil {
			logger.Error(statusErr, FailedStatusError)
		}
		return ctrl.Result{RequeueAfter: FailedRequeueDuration}, err
	}

	if r.Recorder != nil && assignment.Status.AssignmentID != "" {
		if !wasAssigned {
			r.Recorder.Eventf(assignment, corev1.EventTypeNormal, "PolicyAssignmentCreated", "Created policy assignment %q", assignment.Status.AssignmentID)
		} else {
			r.Recorder.Eventf(assignment, corev1.EventTypeNormal, "PolicyAssignmentUpdated", "Updated policy assignment %q", assignment.Status.AssignmentID)
		}
	}

	r.setCondition(assignment, metav1.ConditionTrue, "Reconciled", "Policy assignment successfully reconciled")
	if err := r.Status().Update(ctx, assignment); err != nil {
		logger.Error(err, FailedStatusError)
		return ctrl.Result{RequeueAfter: FailedRequeueDuration}, err
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, nil
}

func (r *AzurePolicyAssignmentReconciler) setCondition(assignment *governancev1alpha1.AzurePolicyAssignment, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&assignment.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: assignment.Generation,
	})
}

func (r *AzurePolicyAssignmentReconciler) setImportedCondition(assignment *governancev1alpha1.AzurePolicyAssignment, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&assignment.Status.Conditions, metav1.Condition{
		Type:               "Imported",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: assignment.Generation,
	})
}

func (r *AzurePolicyAssignmentReconciler) setDriftCondition(assignment *governancev1alpha1.AzurePolicyAssignment, driftFields []string) {
	if len(driftFields) > 0 {
		apimeta.SetStatusCondition(&assignment.Status.Conditions, metav1.Condition{
			Type:               "DriftDetected",
			Status:             metav1.ConditionTrue,
			Reason:             "SpecMismatch",
			Message:            fmt.Sprintf("Live Azure assignment differs from desired spec: %s", strings.Join(driftFields, ", ")),
			ObservedGeneration: assignment.Generation,
		})
	} else {
		apimeta.SetStatusCondition(&assignment.Status.Conditions, metav1.Condition{
			Type:               "DriftDetected",
			Status:             metav1.ConditionFalse,
			Reason:             "InSync",
			Message:            "Azure assignment matches desired spec.",
			ObservedGeneration: assignment.Generation,
		})
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *AzurePolicyAssignmentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.Recorder = mgr.GetEventRecorderFor("azurepolicyassignment-controller")

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
