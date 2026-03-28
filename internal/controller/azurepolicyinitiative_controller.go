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
)

const azurePolicyInitiativeFinalizer = "governance.platform.io/azurepolicyinitiative-finalizer"

// InitiativeService is the interface for managing Azure Policy Set Definitions.
type InitiativeService interface {
	CreateOrUpdate(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedPolicyDefinitionIDs []string) (string, error)
	Delete(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative) error
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
	logger := log.FromContext(ctx)

	initiative := &governancev1alpha1.AzurePolicyInitiative{}
	if err := r.Get(ctx, req.NamespacedName, initiative); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Handle deletion
	if !initiative.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(initiative, azurePolicyInitiativeFinalizer) {
			logger.Info("Running finalizer cleanup", "name", initiative.Name)
			if initiative.Status.InitiativeID != "" {
				if err := r.Service.Delete(ctx, initiative); err != nil {
					r.setCondition(initiative, "Ready", metav1.ConditionFalse, "DeleteFailed", err.Error())
					if statusErr := r.Status().Update(ctx, initiative); statusErr != nil {
						logger.Error(statusErr, FailedStatusError)
					}
					return ctrl.Result{}, err
				}
			}
			controllerutil.RemoveFinalizer(initiative, azurePolicyInitiativeFinalizer)
			if err := r.Update(ctx, initiative); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(initiative, azurePolicyInitiativeFinalizer) {
		controllerutil.AddFinalizer(initiative, azurePolicyInitiativeFinalizer)
		if err := r.Update(ctx, initiative); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// Resolve all policyDefinitionRef entries to Azure resource IDs
	resolvedIDs, err := r.resolvePolicyDefinitionIDs(ctx, initiative, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}
	if resolvedIDs == nil {
		// A ref was not ready yet; status already updated, requeue
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	// Create or update the Azure Policy Set Definition
	initiativeID, err := r.Service.CreateOrUpdate(ctx, initiative, resolvedIDs)
	if err != nil {
		logger.Error(err, "failed to create/update policy initiative")
		r.setCondition(initiative, "Ready", metav1.ConditionFalse, "ReconcileFailed", err.Error())
		if statusErr := r.Status().Update(ctx, initiative); statusErr != nil {
			logger.Error(statusErr, FailedStatusError)
		}
		return ctrl.Result{}, err
	}

	initiative.Status.InitiativeID = initiativeID
	initiative.Status.AppliedVersion = initiative.Spec.Version
	r.setCondition(initiative, "Ready", metav1.ConditionTrue, "Reconciled", "Policy initiative successfully reconciled")
	if err := r.Status().Update(ctx, initiative); err != nil {
		logger.Error(err, FailedStatusError)
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: DefaultRequeueDuration}, nil
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
				r.setCondition(initiative, "Ready", metav1.ConditionFalse, "RefNotFound",
					fmt.Sprintf("AzurePolicyDefinition %q not found: %v", ref.PolicyDefinitionRef, err))
				_ = r.Status().Update(ctx, initiative)
				return nil, err
			}
			if policyDef.Status.PolicyDefinitionID == "" {
				r.setCondition(initiative, "Ready", metav1.ConditionFalse, "RefNotReady",
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

func (r *AzurePolicyInitiativeReconciler) setCondition(initiative *governancev1alpha1.AzurePolicyInitiative, condType string, status metav1.ConditionStatus, reason, message string) {
	apimeta.SetStatusCondition(&initiative.Status.Conditions, metav1.Condition{
		Type:               condType,
		Status:             status,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: initiative.Generation,
	})
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
