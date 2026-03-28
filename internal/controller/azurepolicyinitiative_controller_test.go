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
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	governancev1alpha1 "github.com/vimal-vijayan/azure-policy-operator/api/v1alpha1"
)

// fakeInitiativeService is a test double for the InitiativeService interface.
type fakeInitiativeService struct {
	createOrUpdateFn func(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedIDs []string) (string, error)
	deleteFn         func(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative) error
}

func (f *fakeInitiativeService) CreateOrUpdate(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative, resolvedIDs []string) (string, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, initiative, resolvedIDs)
	}
	return "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policySetDefinitions/" + initiative.Name, nil
}

func (f *fakeInitiativeService) Delete(ctx context.Context, initiative *governancev1alpha1.AzurePolicyInitiative) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, initiative)
	}
	return nil
}

var _ = Describe("AzurePolicyInitiative Controller", func() {
	const (
		resourceName          = "test-initiative"
		initiativeDisplayName = "Test Initiative"
		refDefDisplayName     = "Ref Def"
		fakePolicyID          = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/test-policy"
		fakeInitiativeID      = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policySetDefinitions/test-initiative"
	)

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"}

	newResource := func(withFinalizer bool, policyDefs ...governancev1alpha1.PolicyDefinitionReference) *governancev1alpha1.AzurePolicyInitiative {
		if len(policyDefs) == 0 {
			policyDefs = []governancev1alpha1.PolicyDefinitionReference{
				{PolicyDefinitionID: fakePolicyID},
			}
		}
		res := &governancev1alpha1.AzurePolicyInitiative{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			Spec: governancev1alpha1.AzurePolicyInitiativeSpec{
				DisplayName:       initiativeDisplayName,
				PolicyDefinitions: policyDefs,
			},
		}
		if withFinalizer {
			res.Finalizers = []string{azurePolicyInitiativeFinalizer}
		}
		return res
	}

	newReconciler := func(svc InitiativeService) *AzurePolicyInitiativeReconciler {
		return &AzurePolicyInitiativeReconciler{
			Client:  k8sClient,
			Scheme:  k8sClient.Scheme(),
			Service: svc,
		}
	}

	cleanupResource := func() {
		res := &governancev1alpha1.AzurePolicyInitiative{}
		if err := k8sClient.Get(ctx, namespacedName, res); err == nil {
			res.Finalizers = nil
			_ = k8sClient.Update(ctx, res)
			_ = k8sClient.Delete(ctx, res)
		}
	}

	Context("When reconciling a new resource without a finalizer", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, newResource(false))).To(Succeed())
		})
		AfterEach(func() { cleanupResource() })

		It("should add the finalizer without calling CreateOrUpdate", func() {
			createCalled := false
			svc := &fakeInitiativeService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative, _ []string) (string, error) {
					createCalled = true
					return "", nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(createCalled).To(BeFalse(), "CreateOrUpdate should not be called on first reconcile")

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyInitiativeFinalizer))
		})
	})

	Context("When reconciling a resource with direct policyDefinitionId entries", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, newResource(true))).To(Succeed())
		})
		AfterEach(func() { cleanupResource() })

		It("should call CreateOrUpdate with resolved IDs and set InitiativeID and Ready=True", func() {
			var capturedIDs []string
			svc := &fakeInitiativeService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative, resolvedIDs []string) (string, error) {
					capturedIDs = resolvedIDs
					return fakeInitiativeID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedIDs).To(Equal([]string{fakePolicyID}))

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.InitiativeID).To(Equal(fakeInitiativeID))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal("Reconciled"))
		})

		It("should record AppliedVersion in status after successful reconciliation", func() {
			res := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, res)).To(Succeed())
			res.Spec.Version = "2.0.0"
			Expect(k8sClient.Update(ctx, res)).To(Succeed())

			svc := &fakeInitiativeService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative, _ []string) (string, error) {
					return fakeInitiativeID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.AppliedVersion).To(Equal("2.0.0"))
		})

		It("should set Ready=False with ReconcileFailed reason when CreateOrUpdate fails", func() {
			svc := &fakeInitiativeService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative, _ []string) (string, error) {
					return "", errors.New("azure api error")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("ReconcileFailed"))
			Expect(cond.Message).To(ContainSubstring("azure api error"))
		})
	})

	Context("When reconciling a resource with policyDefinitionRef entries", func() {
		const definitionName = "test-ref-def"

		AfterEach(func() {
			cleanupResource()
			def := &governancev1alpha1.AzurePolicyDefinition{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: definitionName, Namespace: "default"}, def); err == nil {
				def.Finalizers = nil
				_ = k8sClient.Update(ctx, def)
				_ = k8sClient.Delete(ctx, def)
			}
		})

		It("should set Ready=False/RefNotReady and requeue when referenced definition has no PolicyDefinitionID", func() {
			def := &governancev1alpha1.AzurePolicyDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: definitionName, Namespace: "default"},
				Spec:       governancev1alpha1.AzurePolicyDefinitionSpec{DisplayName: "Ref Def", Mode: "All", PolicyRuleJSON: "{}"},
			}
			Expect(k8sClient.Create(ctx, def)).To(Succeed())

			resource := newResource(true, governancev1alpha1.PolicyDefinitionReference{
				PolicyDefinitionRef: definitionName,
			})
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			createCalled := false
			svc := &fakeInitiativeService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative, _ []string) (string, error) {
					createCalled = true
					return "", nil
				},
			}

			result, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			Expect(createCalled).To(BeFalse())

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Reason).To(Equal("RefNotReady"))
		})

		It("should resolve the ref and pass the Azure ID to CreateOrUpdate when definition is ready", func() {
			def := &governancev1alpha1.AzurePolicyDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: definitionName, Namespace: "default"},
				Spec:       governancev1alpha1.AzurePolicyDefinitionSpec{DisplayName: "Ref Def", Mode: "All", PolicyRuleJSON: "{}"},
			}
			Expect(k8sClient.Create(ctx, def)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: definitionName, Namespace: "default"}, def)).To(Succeed())
			def.Status.PolicyDefinitionID = fakePolicyID
			Expect(k8sClient.Status().Update(ctx, def)).To(Succeed())

			resource := newResource(true, governancev1alpha1.PolicyDefinitionReference{
				PolicyDefinitionRef: definitionName,
			})
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			var capturedIDs []string
			svc := &fakeInitiativeService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative, resolvedIDs []string) (string, error) {
					capturedIDs = resolvedIDs
					return fakeInitiativeID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedIDs).To(Equal([]string{fakePolicyID}))

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.InitiativeID).To(Equal(fakeInitiativeID))
		})

		It("should resolve mixed direct IDs and refs in order", func() {
			const secondDefName = "test-ref-def-2"
			const secondFakePolicyID = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/test-policy-2"

			def := &governancev1alpha1.AzurePolicyDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: definitionName, Namespace: "default"},
				Spec:       governancev1alpha1.AzurePolicyDefinitionSpec{DisplayName: "Ref Def", Mode: "All", PolicyRuleJSON: "{}"},
			}
			Expect(k8sClient.Create(ctx, def)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: definitionName, Namespace: "default"}, def)).To(Succeed())
			def.Status.PolicyDefinitionID = fakePolicyID
			Expect(k8sClient.Status().Update(ctx, def)).To(Succeed())

			DeferCleanup(func() {
				d := &governancev1alpha1.AzurePolicyDefinition{}
				if err := k8sClient.Get(ctx, types.NamespacedName{Name: secondDefName, Namespace: "default"}, d); err == nil {
					d.Finalizers = nil
					_ = k8sClient.Update(ctx, d)
					_ = k8sClient.Delete(ctx, d)
				}
			})

			resource := newResource(true,
				governancev1alpha1.PolicyDefinitionReference{PolicyDefinitionRef: definitionName},
				governancev1alpha1.PolicyDefinitionReference{PolicyDefinitionID: secondFakePolicyID},
			)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			var capturedIDs []string
			svc := &fakeInitiativeService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative, resolvedIDs []string) (string, error) {
					capturedIDs = resolvedIDs
					return fakeInitiativeID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedIDs).To(Equal([]string{fakePolicyID, secondFakePolicyID}))
		})
	})

	Context("When deleting a resource with an InitiativeID set in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should call Delete and remove the finalizer", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.InitiativeID = fakeInitiativeID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			deleteCalled := false
			svc := &fakeInitiativeService{
				deleteFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative) error {
					deleteCalled = true
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteCalled).To(BeTrue())

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})

		It("should return an error and keep the finalizer when Delete fails", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.InitiativeID = fakeInitiativeID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			svc := &fakeInitiativeService{
				deleteFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative) error {
					return errors.New("delete failed")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyInitiativeFinalizer))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("DeleteFailed"))
		})
	})

	Context("When deleting a resource without an InitiativeID in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should remove the finalizer without calling Delete", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			deleteCalled := false
			svc := &fakeInitiativeService{
				deleteFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyInitiative) error {
					deleteCalled = true
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteCalled).To(BeFalse())

			updated := &governancev1alpha1.AzurePolicyInitiative{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})
	})

	Context("When the resource does not exist", func() {
		It("should return no error", func() {
			svc := &fakeInitiativeService{}
			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "non-existent", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
