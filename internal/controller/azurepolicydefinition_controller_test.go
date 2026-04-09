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

// fakePolicyDefinitionService is a test double for the DefinitionService interface.
type fakePolicyDefinitionService struct {
	getFn            func(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (string, error)
	createOrUpdateFn func(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (string, error)
	deleteFn         func(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) error
	importFn         func(ctx context.Context, importID string, def *governancev1alpha1.AzurePolicyDefinition) ([]string, error)
}

func (f *fakePolicyDefinitionService) Get(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (string, error) {
	if f.getFn != nil {
		return f.getFn(ctx, def)
	}
	return "", nil
}

func (f *fakePolicyDefinitionService) CreateOrUpdate(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) (string, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, def)
	}
	return "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/" + def.Name, nil
}

func (f *fakePolicyDefinitionService) Delete(ctx context.Context, def *governancev1alpha1.AzurePolicyDefinition) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, def)
	}
	return nil
}

func (f *fakePolicyDefinitionService) Import(ctx context.Context, importID string, def *governancev1alpha1.AzurePolicyDefinition) ([]string, error) {
	if f.importFn != nil {
		return f.importFn(ctx, importID, def)
	}
	return nil, nil
}

var _ = Describe("AzurePolicyDefinition Controller", func() {
	const resourceName = "test-policy-definition"
	const fakeID = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/test-policy-definition"

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"}

	newResource := func(withFinalizer bool) *governancev1alpha1.AzurePolicyDefinition {
		res := &governancev1alpha1.AzurePolicyDefinition{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			Spec: governancev1alpha1.AzurePolicyDefinitionSpec{
				DisplayName:    "Test Policy",
				Mode:           "All",
				PolicyType:     "Custom",
				SubscriptionID: "00000000-0000-0000-0000-000000000000",
				PolicyRuleJSON: "{}",
			},
		}
		if withFinalizer {
			res.Finalizers = []string{azurePolicyDefinitionFinalizer}
		}
		return res
	}

	newReconciler := func(svc DefinitionService) *AzurePolicyDefinitionReconciler {
		return &AzurePolicyDefinitionReconciler{
			Client:  k8sClient,
			Scheme:  k8sClient.Scheme(),
			Service: svc,
		}
	}

	cleanupResource := func() {
		res := &governancev1alpha1.AzurePolicyDefinition{}
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
			svc := &fakePolicyDefinitionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyDefinition) (string, error) {
					createCalled = true
					return "", nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(createCalled).To(BeFalse(), "CreateOrUpdate should not be called on first reconcile")

			updated := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyDefinitionFinalizer))
		})
	})

	Context("When reconciling a resource that already has the finalizer", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, newResource(true))).To(Succeed())
		})
		AfterEach(func() { cleanupResource() })

		It("should call CreateOrUpdate and set PolicyDefinitionID and Ready=True", func() {
			svc := &fakePolicyDefinitionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyDefinition) (string, error) {
					return fakeID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.PolicyDefinitionID).To(Equal(fakeID))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal("Reconciled"))
		})

		It("should record AppliedVersion in status after successful reconciliation", func() {
			res := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8sClient.Get(ctx, namespacedName, res)).To(Succeed())
			res.Spec.Version = "1.0.0"
			Expect(k8sClient.Update(ctx, res)).To(Succeed())

			svc := &fakePolicyDefinitionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyDefinition) (string, error) {
					return fakeID, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.AppliedVersion).To(Equal("1.0.0"))
		})

		It("should set Ready=False with ReconcileFailed reason when CreateOrUpdate fails", func() {
			svc := &fakePolicyDefinitionService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyDefinition) (string, error) {
					return "", errors.New("azure api error")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("ReconcileFailed"))
			Expect(cond.Message).To(ContainSubstring("azure api error"))
		})
	})

	Context("When deleting a resource with a PolicyDefinitionID set in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should call Delete and remove the finalizer", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.PolicyDefinitionID = fakeID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			deleteCalled := false
			svc := &fakePolicyDefinitionService{
				deleteFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyDefinition) error {
					deleteCalled = true
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteCalled).To(BeTrue())

			updated := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})

		It("should return an error and keep the finalizer when Delete fails", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.PolicyDefinitionID = fakeID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			svc := &fakePolicyDefinitionService{
				deleteFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyDefinition) error {
					return errors.New("delete failed")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyDefinitionFinalizer))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("DeleteFailed"))
		})
	})

	Context("When deleting a resource without a PolicyDefinitionID in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should remove the finalizer without calling Delete", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			deleteCalled := false
			svc := &fakePolicyDefinitionService{
				deleteFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyDefinition) error {
					deleteCalled = true
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteCalled).To(BeFalse())

			updated := &governancev1alpha1.AzurePolicyDefinition{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})
	})

	Context("When the resource does not exist", func() {
		It("should return no error", func() {
			svc := &fakePolicyDefinitionService{}
			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "non-existent", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
