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

// fakeExemptionService is a test double for the ExemptionService interface.
type fakeExemptionService struct {
	createOrUpdateFn func(ctx context.Context, exemption *governancev1alpha1.AzurePolicyExemption) (string, error)
	deleteFn         func(ctx context.Context, scope, exemptionID string) error
}

func (f *fakeExemptionService) CreateOrUpdate(ctx context.Context, exemption *governancev1alpha1.AzurePolicyExemption) (string, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, exemption)
	}
	return "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Authorization/policyExemptions/" + exemption.Name, nil
}

func (f *fakeExemptionService) Delete(ctx context.Context, scope, exemptionID string) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, scope, exemptionID)
	}
	return nil
}

var _ = Describe("AzurePolicyExemption Controller", func() {
	const resourceName = "test-exemption"
	const exemptionDisplayName = "Test Exemption"
	const fakeScope = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg"
	const fakeExemptionID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Authorization/policyExemptions/test-exemption"
	const fakeAssignmentID = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Authorization/policyAssignments/test-assignment"

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"}

	newResource := func(withFinalizer bool) *governancev1alpha1.AzurePolicyExemption {
		res := &governancev1alpha1.AzurePolicyExemption{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			Spec: governancev1alpha1.AzurePolicyExemptionSpec{
				DisplayName:        exemptionDisplayName,
				Scope:              fakeScope,
				ExemptionCategory:  "Waiver",
				PolicyAssignmentID: fakeAssignmentID,
			},
		}
		if withFinalizer {
			res.Finalizers = []string{azurePolicyExemptionFinalizer}
		}
		return res
	}

	newReconciler := func(svc ExemptionService) *AzurePolicyExemptionReconciler {
		return &AzurePolicyExemptionReconciler{
			Client:  k8sClient,
			Scheme:  k8sClient.Scheme(),
			Service: svc,
		}
	}

	cleanupResource := func() {
		res := &governancev1alpha1.AzurePolicyExemption{}
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
			svc := &fakeExemptionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyExemption) (string, error) {
					createCalled = true
					return "", nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(createCalled).To(BeFalse(), "CreateOrUpdate should not be called on first reconcile")

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyExemptionFinalizer))
		})
	})

	Context("When reconciling a resource that already has the finalizer", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, newResource(true))).To(Succeed())
		})
		AfterEach(func() { cleanupResource() })

		It("should call CreateOrUpdate and set ExemptionID and Ready=True", func() {
			svc := &fakeExemptionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyExemption) (string, error) {
					return fakeExemptionID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.ExemptionID).To(Equal(fakeExemptionID))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal("Reconciled"))
		})

		It("should set Ready=False with ReconcileFailed reason when CreateOrUpdate fails", func() {
			svc := &fakeExemptionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyExemption) (string, error) {
					return "", errors.New("azure api error")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("ReconcileFailed"))
			Expect(cond.Message).To(ContainSubstring("azure api error"))
		})
	})

	Context("When reconciling a resource with a policyAssignmentRef", func() {
		const assignmentName = "test-assignment-ref"

		AfterEach(func() {
			cleanupResource()
			assignment := &governancev1alpha1.AzurePolicyAssignment{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: assignmentName, Namespace: "default"}, assignment); err == nil {
				assignment.Finalizers = nil
				_ = k8sClient.Update(ctx, assignment)
				_ = k8sClient.Delete(ctx, assignment)
			}
		})

		It("should requeue when the referenced assignment has no AssignmentID yet", func() {
			assignment := &governancev1alpha1.AzurePolicyAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: assignmentName, Namespace: "default"},
				Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
					DisplayName:        "Test Assignment",
					PolicyDefinitionID: fakeAssignmentID,
					Scope:              fakeScope,
				},
			}
			Expect(k8sClient.Create(ctx, assignment)).To(Succeed())

			res := &governancev1alpha1.AzurePolicyExemption{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  "default",
					Finalizers: []string{azurePolicyExemptionFinalizer},
				},
				Spec: governancev1alpha1.AzurePolicyExemptionSpec{
					DisplayName:         exemptionDisplayName,
					Scope:               fakeScope,
					ExemptionCategory:   "Waiver",
					PolicyAssignmentRef: assignmentName,
				},
			}
			Expect(k8sClient.Create(ctx, res)).To(Succeed())

			createCalled := false
			svc := &fakeExemptionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyExemption) (string, error) {
					createCalled = true
					return "", nil
				},
			}

			result, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Requeue).To(BeTrue())
			Expect(createCalled).To(BeFalse())

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Reason).To(Equal("RefNotReady"))
		})

		It("should resolve policyAssignmentRef and call CreateOrUpdate when assignment is ready", func() {
			assignment := &governancev1alpha1.AzurePolicyAssignment{
				ObjectMeta: metav1.ObjectMeta{Name: assignmentName, Namespace: "default"},
				Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
					DisplayName:        "Test Assignment",
					PolicyDefinitionID: fakeAssignmentID,
					Scope:              fakeScope,
				},
			}
			Expect(k8sClient.Create(ctx, assignment)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: assignmentName, Namespace: "default"}, assignment)).To(Succeed())
			assignment.Status.AssignmentID = fakeAssignmentID
			Expect(k8sClient.Status().Update(ctx, assignment)).To(Succeed())

			res := &governancev1alpha1.AzurePolicyExemption{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  "default",
					Finalizers: []string{azurePolicyExemptionFinalizer},
				},
				Spec: governancev1alpha1.AzurePolicyExemptionSpec{
					DisplayName:         exemptionDisplayName,
					Scope:               fakeScope,
					ExemptionCategory:   "Waiver",
					PolicyAssignmentRef: assignmentName,
				},
			}
			Expect(k8sClient.Create(ctx, res)).To(Succeed())

			svc := &fakeExemptionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyExemption) (string, error) {
					return fakeExemptionID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.ExemptionID).To(Equal(fakeExemptionID))
		})
	})

	Context("When deleting a resource with an ExemptionID set in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should call Delete and remove the finalizer", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.ExemptionID = fakeExemptionID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			var deletedScope, deletedID string
			svc := &fakeExemptionService{
				deleteFn: func(_ context.Context, scope, exemptionID string) error {
					deletedScope = scope
					deletedID = exemptionID
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deletedScope).To(Equal(fakeScope))
			Expect(deletedID).To(Equal(fakeExemptionID))

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})

		It("should return an error and keep the finalizer when Delete fails", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.ExemptionID = fakeExemptionID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			svc := &fakeExemptionService{
				deleteFn: func(_ context.Context, _, _ string) error {
					return errors.New("delete failed")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyExemptionFinalizer))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("DeleteFailed"))
		})
	})

	Context("When deleting a resource without an ExemptionID in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should remove the finalizer without calling Delete", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			deleteCalled := false
			svc := &fakeExemptionService{
				deleteFn: func(_ context.Context, _, _ string) error {
					deleteCalled = true
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteCalled).To(BeFalse())

			updated := &governancev1alpha1.AzurePolicyExemption{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})
	})

	Context("When the resource does not exist", func() {
		It("should return no error", func() {
			svc := &fakeExemptionService{}
			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "non-existent", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
