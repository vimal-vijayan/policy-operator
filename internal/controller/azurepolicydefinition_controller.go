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
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
)

const (
	azurePolicyDefinitionFinalizer = "governance.platform.io/azurepolicydefinition-finalizer"
)

// DefinitionService is the interface for managing Azure Policy Definitions.
type DefinitionService interface {
	CreateOrUpdate(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (string, error)
	Delete(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) error
	Import(ctx context.Context, importID string, def *governancev1alpha1.AzurePolicyDefinition) ([]string, error)
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

	result, done, err := r.handleImport(ctx, policyDef)
	if done {
		return result, err
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, r.reconcileDefinition(ctx, policyDef)
}

func (r *AzurePolicyDefinitionReconciler) handleDeletion(ctx context.Context, policyDef *governancev1alpha1.AzurePolicyDefinition) error {
	logger := log.FromContext(ctx)

	if !controllerutil.ContainsFinalizer(policyDef, azurePolicyDefinitionFinalizer) {
		return nil
	}

	logger.Info("Running finalizer cleanup", "name", policyDef.Name)

	importMode := policyDef.Annotations[annotationImportMode]

	if importMode == importModeObserveOnly {
		logger.Info("Skipping finalizer cleanup due to observe-only import mode", "name", policyDef.Name)
		controllerutil.RemoveFinalizer(policyDef, azurePolicyDefinitionFinalizer)
		return r.Update(ctx, policyDef)
	}

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

	readyReason := "Reconciled"
	readyMsg := "Policy definition successfully reconciled"
	if policyDef.Annotations[annotationImportMode] == importModeOnlyOnce {
		readyReason = "AppliedOnce"
		readyMsg = "Policy definition applied once from import. No further changes will be applied to Azure."
	}
	r.setCondition(policyDef, metav1.ConditionTrue, readyReason, readyMsg)
	return r.Status().Update(ctx, policyDef)
}

func (r *AzurePolicyDefinitionReconciler) handleImport(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (ctrl.Result, bool, error) {
	logger := log.FromContext(ctx)
	importID := def.Annotations[annotationImportID]

	if importID == "" {
		return ctrl.Result{}, false, nil
	}

	// Prevent rebinding when status already points to a different Azure resource ID.
	if def.Status.PolicyDefinitionID != "" && importID != def.Status.PolicyDefinitionID {
		logger.V(1).Error(fmt.Errorf("The status policyDefinitionId %q does not match the provided import-id annotation %q", def.Status.PolicyDefinitionID, importID), "Import ID mismatch")
		msg := fmt.Sprintf("annotation import-id %q differs from already bound policyDefinitionId %q", importID, def.Status.PolicyDefinitionID)
		r.setCondition(def, metav1.ConditionFalse, "ImportConflict", msg)
		_ = r.Status().Update(ctx, def)
		return ctrl.Result{}, true, fmt.Errorf("import conflict: %s", msg)
	}

	importMode := def.Annotations[annotationImportMode]
	if importMode == "" {
		importMode = importModeObserveOnly
	}

	// For "adopt-once" mode: if already applied, skip re-reconciling Azure.
	if importMode == importModeOnlyOnce {
		cond := apimeta.FindStatusCondition(def.Status.Conditions, "Ready")
		if cond != nil && cond.Reason == "AppliedOnce" {
			logger.V(1).Info("Skipping reconcile — already applied once", "name", def.Name)
			return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, true, nil
		}
	}

	logger.Info("Importing existing Azure Policy Definition", "importID", importID, "importMode", importMode)

	driftFields, err := r.Service.Import(ctx, importID, def)
	if err != nil {
		r.setImportedCondition(def, metav1.ConditionFalse, "ImportFailed", err.Error())
		r.setCondition(def, metav1.ConditionFalse, "ImportFailed", err.Error())
		_ = r.Status().Update(ctx, def)
		return ctrl.Result{RequeueAfter: 3 * time.Minute}, true, err
	}

	def.Status.PolicyDefinitionID = importID
	r.setImportedCondition(def, metav1.ConditionTrue, "ImportSucceeded", "Existing Azure Policy Definition was adopted successfully.")
	r.setDriftCondition(def, driftFields)

	if importMode == importModeObserveOnly {
		r.setCondition(def, metav1.ConditionTrue, "ObservedOnly", "Resource imported in observe-only mode. No changes applied to Azure.")
		if err := r.Status().Update(ctx, def); err != nil {
			logger.Error(err, FailedStatusError)
			return ctrl.Result{}, true, err
		}
		return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, true, nil
	}

	// For "adopt-once" and "reconcile" modes: fall through to CreateOrUpdate.
	return ctrl.Result{}, false, nil
}

func (r *AzurePolicyDefinitionReconciler) setImportedCondition(def *governancev1alpha1.AzurePolicyDefinition, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&def.Status.Conditions, metav1.Condition{
		Type:               "Imported",
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: def.Generation,
	})
}

func (r *AzurePolicyDefinitionReconciler) setDriftCondition(def *governancev1alpha1.AzurePolicyDefinition, driftFields []string) {
	if len(driftFields) > 0 {
		apimeta.SetStatusCondition(&def.Status.Conditions, metav1.Condition{
			Type:               "DriftDetected",
			Status:             metav1.ConditionTrue,
			Reason:             "SpecMismatch",
			Message:            fmt.Sprintf("Live Azure definition differs from desired spec: %s", strings.Join(driftFields, ", ")),
			ObservedGeneration: def.Generation,
		})
	} else {
		apimeta.SetStatusCondition(&def.Status.Conditions, metav1.Condition{
			Type:               "DriftDetected",
			Status:             metav1.ConditionFalse,
			Reason:             "InSync",
			Message:            "Azure definition matches desired spec.",
			ObservedGeneration: def.Generation,
		})
	}
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
