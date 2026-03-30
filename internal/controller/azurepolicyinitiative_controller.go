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
)

const azurePolicyInitiativeFinalizer = "governance.platform.io/azurepolicyinitiative-finalizer"

// InitiativeService is the interface for managing Azure Policy Set Definitions.
type InitiativeService interface {
	CreateOrUpdate(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedPolicyDefinitionIDs []string) (string, error)
	Delete(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative) error
	Import(ctx context.Context, importID string, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedPolicyDefinitionIDs []string) ([]string, error)
}

// AzurePolicyInitiativeReconciler reconciles a AzurePolicyInitiative object
type AzurePolicyInitiativeReconciler struct {
	client.Client
	Scheme  *runtime.Scheme
	Service InitiativeService
}

// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyinitiatives,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyinitiatives/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicyinitiatives/finalizers,verbs=update
// +kubebuilder:rbac:groups=governance.platform.io,resources=azurepolicydefinitions,verbs=get;list;watch

func (r *AzurePolicyInitiativeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	initiative := &governancev1alpha1.AzurePolicyInitiative{}
	if err := r.Get(ctx, req.NamespacedName, initiative); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !initiative.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, r.handleDeletion(ctx, initiative)
	}

	if !controllerutil.ContainsFinalizer(initiative, azurePolicyInitiativeFinalizer) {
		controllerutil.AddFinalizer(initiative, azurePolicyInitiativeFinalizer)
		return ctrl.Result{}, r.Update(ctx, initiative)
	}

	resolvedIDs, err := r.resolvePolicyDefinitionIDs(ctx, initiative, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	if resolvedIDs == nil {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	result, done, err := r.handleImport(ctx, initiative, resolvedIDs)
	if done {
		return result, err
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, r.reconcileInitiative(ctx, initiative, resolvedIDs)
}

func (r *AzurePolicyInitiativeReconciler) handleDeletion(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative) error {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(initiative, azurePolicyInitiativeFinalizer) {
		return nil
	}

	logger.Info("Running finalizer cleanup", "name", initiative.Name)

	if initiative.Annotations[annotationImportMode] == importModeObserveOnly {
		logger.Info("Skipping finalizer cleanup due to observe-only import mode", "name", initiative.Name)
		controllerutil.RemoveFinalizer(initiative, azurePolicyInitiativeFinalizer)
		return r.Update(ctx, initiative)
	}

	if initiative.Status.InitiativeID != "" {
		if err := r.Service.Delete(ctx, initiative); err != nil {
			r.setCondition(initiative, metav1.ConditionFalse, "DeleteFailed", err.Error())
			if statusErr := r.Status().Update(ctx, initiative); statusErr != nil {
				logger.Error(statusErr, FailedStatusError)
			}
			return err
		}
	}

	controllerutil.RemoveFinalizer(initiative, azurePolicyInitiativeFinalizer)
	return r.Update(ctx, initiative)
}

func (r *AzurePolicyInitiativeReconciler) reconcileInitiative(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedIDs []string) error {
	logger := log.FromContext(ctx)

	initiativeID, err := r.Service.CreateOrUpdate(ctx, initiative, resolvedIDs)
	if err != nil {
		logger.Error(err, "failed to create/update policy initiative")
		r.setCondition(initiative, metav1.ConditionFalse, "ReconcileFailed", err.Error())
		if statusErr := r.Status().Update(ctx, initiative); statusErr != nil {
			logger.Error(statusErr, FailedStatusError)
		}
		return err
	}

	initiative.Status.InitiativeID = initiativeID
	initiative.Status.AppliedVersion = initiative.Spec.Version

	readyReason := "Reconciled"
	readyMsg := "Policy initiative successfully reconciled"
	if initiative.Annotations[annotationImportMode] == importModeOnlyOnce {
		readyReason = ReasonAppliedOnce
		readyMsg = "Policy initiative applied once from import. No further changes will be applied to Azure."
	}
	r.setCondition(initiative, metav1.ConditionTrue, readyReason, readyMsg)
	return r.Status().Update(ctx, initiative)
}

func (r *AzurePolicyInitiativeReconciler) handleImport(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedIDs []string) (ctrl.Result, bool, error) {
	logger := log.FromContext(ctx)
	importID := initiative.Annotations[annotationImportID]

	if importID == "" {
		return ctrl.Result{}, false, nil
	}

	if initiative.Status.InitiativeID != "" && !strings.EqualFold(importID, initiative.Status.InitiativeID) {
		msg := fmt.Sprintf("annotation import-id %q differs from already bound initiativeId %q", importID, initiative.Status.InitiativeID)
		r.setCondition(initiative, metav1.ConditionFalse, "ImportConflict", msg)
		_ = r.Status().Update(ctx, initiative)
		return ctrl.Result{}, true, fmt.Errorf("import conflict: %s", msg)
	}

	importMode := initiative.Annotations[annotationImportMode]
	if importMode == "" {
		importMode = importModeObserveOnly
	}

	if importMode == importModeOnlyOnce {
		cond := apimeta.FindStatusCondition(initiative.Status.Conditions, "Ready")
		if cond != nil && cond.Reason == ReasonAppliedOnce {
			logger.V(1).Info("Skipping reconcile - already applied once", "name", initiative.Name)
			return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, true, nil
		}
	}

	logger.Info("Importing existing Azure Policy Set Definition", "importID", importID, "importMode", importMode)

	driftFields, err := r.Service.Import(ctx, importID, initiative, resolvedIDs)
	if err != nil {
		r.setImportedCondition(initiative, metav1.ConditionFalse, "ImportFailed", err.Error())
		r.setCondition(initiative, metav1.ConditionFalse, "ImportFailed", err.Error())
		_ = r.Status().Update(ctx, initiative)
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, true, err
	}

	initiative.Status.InitiativeID = importID
	r.setImportedCondition(initiative, metav1.ConditionTrue, "ImportSucceeded", "Existing Azure Policy Set Definition was adopted successfully.")
	r.setDriftCondition(initiative, driftFields)

	if importMode == importModeObserveOnly {
		r.setCondition(initiative, metav1.ConditionTrue, "ObservedOnly", "Resource imported in observe-only mode. No changes applied to Azure.")
		if err := r.Status().Update(ctx, initiative); err != nil {
			logger.Error(err, FailedStatusError)
			return ctrl.Result{}, true, err
		}
		return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, true, nil
	}

	return ctrl.Result{}, false, nil
}

// resolvePolicyDefinitionIDs resolves all policyDefinitionRef entries to Azure resource IDs.
// Returns nil (with status updated) if any ref is not ready, otherwise returns the resolved IDs
// in order matching spec.policyDefinitions.
func (r *AzurePolicyInitiativeReconciler) resolvePolicyDefinitionIDs(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, namespace string) ([]string, error) {
	resolved := make([]string, len(initiative.Spec.PolicyDefinitions))
	for i, ref := range initiative.Spec.PolicyDefinitions {
		if ref.PolicyDefinitionRef != "" {
			policyDef := &governancev1alpha1.AzurePolicyDefinition{}
			if err := r.Get(ctx, types.NamespacedName{Name: ref.PolicyDefinitionRef, Namespace: namespace}, policyDef); err != nil {
				r.setCondition(initiative, metav1.ConditionFalse, "RefNotFound",
					fmt.Sprintf("AzurePolicyDefinition %q not found: %v", ref.PolicyDefinitionRef, err))
				_ = r.Status().Update(ctx, initiative)
				return nil, err
			}
			if policyDef.Status.PolicyDefinitionID == "" {
				r.setCondition(initiative, metav1.ConditionFalse, "RefNotReady",
					fmt.Sprintf("AzurePolicyDefinition %q has no policyDefinitionId in status yet", ref.PolicyDefinitionRef))
				_ = r.Status().Update(ctx, initiative)
				return nil, nil
			}
			resolved[i] = policyDef.Status.PolicyDefinitionID
		} else {
			resolved[i] = ref.PolicyDefinitionID
		}
	}
	return resolved, nil
}

func (r *AzurePolicyInitiativeReconciler) setCondition(initiative *governancev1alpha1.AzurePolicyInitiative, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&initiative.Status.Conditions, metav1.Condition{
		Type:               "Ready",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: initiative.Generation,
	})
}

func (r *AzurePolicyInitiativeReconciler) setImportedCondition(initiative *governancev1alpha1.AzurePolicyInitiative, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&initiative.Status.Conditions, metav1.Condition{
		Type:               "Imported",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: initiative.Generation,
	})
}

func (r *AzurePolicyInitiativeReconciler) setDriftCondition(initiative *governancev1alpha1.AzurePolicyInitiative, driftFields []string) {
	if len(driftFields) > 0 {
		apimeta.SetStatusCondition(&initiative.Status.Conditions, metav1.Condition{
			Type:               "DriftDetected",
			Status:             metav1.ConditionTrue,
			Reason:             "SpecMismatch",
			Message:            fmt.Sprintf("Live Azure initiative differs from desired spec: %s", strings.Join(driftFields, ", ")),
			ObservedGeneration: initiative.Generation,
		})
	} else {
		apimeta.SetStatusCondition(&initiative.Status.Conditions, metav1.Condition{
			Type:               "DriftDetected",
			Status:             metav1.ConditionFalse,
			Reason:             "InSync",
			Message:            "Azure initiative matches desired spec.",
			ObservedGeneration: initiative.Generation,
		})
	}
}

// SetupWithManager sets up the controller with the Manager.
func (r *AzurePolicyInitiativeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&governancev1alpha1.AzurePolicyInitiative{}).
		Watches(
			&governancev1alpha1.AzurePolicyDefinition{},
			handler.EnqueueRequestsFromMapFunc(r.initiativesReferencingDefinition),
		).
		Complete(r)
}

// initiativesReferencingDefinition maps an AzurePolicyDefinition event to all
// AzurePolicyInitiatives that reference it via policyDefinitionRef.
func (r *AzurePolicyInitiativeReconciler) initiativesReferencingDefinition(ctx context.Context, obj client.Object) []reconcile.Request {
	list := &governancev1alpha1.AzurePolicyInitiativeList{}
	if err := r.List(ctx, list, client.InNamespace(obj.GetNamespace())); err != nil {
		return nil
	}
	var requests []reconcile.Request
	for _, initiative := range list.Items {
		for _, ref := range initiative.Spec.PolicyDefinitions {
			if ref.PolicyDefinitionRef == obj.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{Name: initiative.Name, Namespace: initiative.Namespace},
				})
				break
			}
		}
	}
	return requests
}
