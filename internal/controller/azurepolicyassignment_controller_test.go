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

// fakePolicyAssignmentService is a test double for the PolicyAssignmentService interface.
type fakePolicyAssignmentService struct {
	createOrUpdateFn func(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error)
	deleteFn         func(ctx context.Context, scope, assignmentID string, exemptions []governancev1alpha1.AssignmentExemptionStatus, identity *governancev1alpha1.AssignmentIdentity) error
	importFn         func(ctx context.Context, importID string, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, []string, error)
}

func (f *fakePolicyAssignmentService) CreateOrUpdate(ctx context.Context, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
	if f.createOrUpdateFn != nil {
		return f.createOrUpdateFn(ctx, assignment, policyDefinitionID)
	}
	id := "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Authorization/policyAssignments/" + assignment.Name
	return id, "", "", nil, nil
}

func (f *fakePolicyAssignmentService) Delete(ctx context.Context, scope, assignmentID string, exemptions []governancev1alpha1.AssignmentExemptionStatus, identity *governancev1alpha1.AssignmentIdentity) error {
	if f.deleteFn != nil {
		return f.deleteFn(ctx, scope, assignmentID, exemptions, identity)
	}
	return nil
}

func (f *fakePolicyAssignmentService) Import(ctx context.Context, importID string, assignment *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, []string, error) {
	if f.importFn != nil {
		return f.importFn(ctx, importID, assignment, policyDefinitionID)
	}
	return "", "", nil, nil
}

var _ = Describe("AzurePolicyAssignment Controller", func() {
	const (
		resourceName          = "test-assignment"
		assignmentDisplayName = "Test Assignment"
		fakeScope             = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg"
		fakePolicyDefID       = "/subscriptions/00000000-0000-0000-0000-000000000000/providers/Microsoft.Authorization/policyDefinitions/test-def"
		fakeAssignmentID      = "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg/providers/Microsoft.Authorization/policyAssignments/test-assignment"
	)

	ctx := context.Background()
	namespacedName := types.NamespacedName{Name: resourceName, Namespace: "default"}

	newResource := func(withFinalizer bool) *governancev1alpha1.AzurePolicyAssignment {
		res := &governancev1alpha1.AzurePolicyAssignment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: "default",
			},
			Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
				DisplayName:        assignmentDisplayName,
				Scope:              fakeScope,
				PolicyDefinitionID: fakePolicyDefID,
				EnforcementMode:    "Default",
			},
		}
		if withFinalizer {
			res.Finalizers = []string{azurePolicyAssignmentFinalizer}
		}
		return res
	}

	newReconciler := func(svc PolicyAssignmentService) *AzurePolicyAssignmentReconciler {
		return &AzurePolicyAssignmentReconciler{
			Client:  k8sClient,
			Scheme:  k8sClient.Scheme(),
			Service: svc,
		}
	}

	cleanupResource := func() {
		res := &governancev1alpha1.AzurePolicyAssignment{}
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
			svc := &fakePolicyAssignmentService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyAssignment, _ string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
					createCalled = true
					return "", "", "", nil, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(createCalled).To(BeFalse(), "CreateOrUpdate should not be called on first reconcile")

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyAssignmentFinalizer))
		})
	})

	Context("When reconciling a resource with a finalizer and direct policyDefinitionId", func() {
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, newResource(true))).To(Succeed())
		})
		AfterEach(func() { cleanupResource() })

		It("should call CreateOrUpdate with the correct policyDefinitionID and set AssignmentID and Ready=True", func() {
			var capturedDefID string
			svc := &fakePolicyAssignmentService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
					capturedDefID = policyDefinitionID
					return fakeAssignmentID, "westeurope", "principal-id-123", nil, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedDefID).To(Equal(fakePolicyDefID))

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.AssignmentID).To(Equal(fakeAssignmentID))
			Expect(updated.Status.AssignedLocation).To(Equal("westeurope"))
			Expect(updated.Status.MIPrincipalID).To(Equal("principal-id-123"))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal("Reconciled"))
		})

		It("should set Ready=False with ReconcileFailed reason when CreateOrUpdate fails", func() {
			svc := &fakePolicyAssignmentService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyAssignment, _ string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
					return "", "", "", nil, errors.New("azure api error")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("ReconcileFailed"))
			Expect(cond.Message).To(ContainSubstring("azure api error"))
		})

		It("should persist inline exemption statuses returned by CreateOrUpdate", func() {
			exemptions := []governancev1alpha1.AssignmentExemptionStatus{
				{DisplayName: "Exemption A", ExemptionID: "/sub/rg/exemptions/a", Scope: fakeScope},
			}
			svc := &fakePolicyAssignmentService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyAssignment, _ string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
					return fakeAssignmentID, "", "", exemptions, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.Exemptions).To(HaveLen(1))
			Expect(updated.Status.Exemptions[0].DisplayName).To(Equal("Exemption A"))
		})
	})

	Context("When reconciling a resource with a policyDefinitionRef", func() {
		const definitionName = "test-def-ref"

		AfterEach(func() {
			cleanupResource()
			def := &governancev1alpha1.AzurePolicyDefinition{}
			if err := k8sClient.Get(ctx, types.NamespacedName{Name: definitionName, Namespace: "default"}, def); err == nil {
				def.Finalizers = nil
				_ = k8sClient.Update(ctx, def)
				_ = k8sClient.Delete(ctx, def)
			}
		})

		It("should set Ready=False/RefNotReady and requeue when the referenced definition has no PolicyDefinitionID", func() {
			def := &governancev1alpha1.AzurePolicyDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: definitionName, Namespace: "default"},
				Spec: governancev1alpha1.AzurePolicyDefinitionSpec{
					DisplayName:    "Ref Definition",
					Mode:           "All",
					PolicyRuleJSON: "{}",
				},
			}
			Expect(k8sClient.Create(ctx, def)).To(Succeed())

			res := &governancev1alpha1.AzurePolicyAssignment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  "default",
					Finalizers: []string{azurePolicyAssignmentFinalizer},
				},
				Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
					DisplayName:         assignmentDisplayName,
					Scope:               fakeScope,
					PolicyDefinitionRef: definitionName,
					EnforcementMode:     "Default",
				},
			}
			Expect(k8sClient.Create(ctx, res)).To(Succeed())

			createCalled := false
			svc := &fakePolicyAssignmentService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyAssignment, _ string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
					createCalled = true
					return "", "", "", nil, nil
				},
			}

			result, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.RequeueAfter).To(BeNumerically(">", 0))
			Expect(createCalled).To(BeFalse())

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Reason).To(Equal("RefNotReady"))
		})

		It("should resolve the ref and call CreateOrUpdate when the referenced definition is ready", func() {
			def := &governancev1alpha1.AzurePolicyDefinition{
				ObjectMeta: metav1.ObjectMeta{Name: definitionName, Namespace: "default"},
				Spec: governancev1alpha1.AzurePolicyDefinitionSpec{
					DisplayName:    "Ref Definition",
					Mode:           "All",
					PolicyRuleJSON: "{}",
				},
			}
			Expect(k8sClient.Create(ctx, def)).To(Succeed())
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: definitionName, Namespace: "default"}, def)).To(Succeed())
			def.Status.PolicyDefinitionID = fakePolicyDefID
			Expect(k8sClient.Status().Update(ctx, def)).To(Succeed())

			res := &governancev1alpha1.AzurePolicyAssignment{
				ObjectMeta: metav1.ObjectMeta{
					Name:       resourceName,
					Namespace:  "default",
					Finalizers: []string{azurePolicyAssignmentFinalizer},
				},
				Spec: governancev1alpha1.AzurePolicyAssignmentSpec{
					DisplayName:         assignmentDisplayName,
					Scope:               fakeScope,
					PolicyDefinitionRef: definitionName,
					EnforcementMode:     "Default",
				},
			}
			Expect(k8sClient.Create(ctx, res)).To(Succeed())

			var capturedDefID string
			svc := &fakePolicyAssignmentService{
				createOrUpdateFn: func(_ context.Context, _ *governancev1alpha1.AzurePolicyAssignment, policyDefinitionID string) (string, string, string, []governancev1alpha1.AssignmentExemptionStatus, error) {
					capturedDefID = policyDefinitionID
					return fakeAssignmentID, "", "", nil, nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(capturedDefID).To(Equal(fakePolicyDefID))

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Status.AssignmentID).To(Equal(fakeAssignmentID))
		})
	})

	Context("When deleting a resource with an AssignmentID set in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should call Delete with correct args and remove the finalizer", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.AssignmentID = fakeAssignmentID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			var deletedScope, deletedID string
			svc := &fakePolicyAssignmentService{
				deleteFn: func(_ context.Context, scope, assignmentID string, _ []governancev1alpha1.AssignmentExemptionStatus, _ *governancev1alpha1.AssignmentIdentity) error {
					deletedScope = scope
					deletedID = assignmentID
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deletedScope).To(Equal(fakeScope))
			Expect(deletedID).To(Equal(fakeAssignmentID))

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})

		It("should return an error and keep the finalizer when Delete fails", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())

			Expect(k8sClient.Get(ctx, namespacedName, resource)).To(Succeed())
			resource.Status.AssignmentID = fakeAssignmentID
			Expect(k8sClient.Status().Update(ctx, resource)).To(Succeed())

			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			svc := &fakePolicyAssignmentService{
				deleteFn: func(_ context.Context, _, _ string, _ []governancev1alpha1.AssignmentExemptionStatus, _ *governancev1alpha1.AssignmentIdentity) error {
					return errors.New("delete failed")
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).To(HaveOccurred())

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8sClient.Get(ctx, namespacedName, updated)).To(Succeed())
			Expect(updated.Finalizers).To(ContainElement(azurePolicyAssignmentFinalizer))

			cond := apimeta.FindStatusCondition(updated.Status.Conditions, "Ready")
			Expect(cond).NotTo(BeNil())
			Expect(cond.Status).To(Equal(metav1.ConditionFalse))
			Expect(cond.Reason).To(Equal("DeleteFailed"))
		})
	})

	Context("When deleting a resource without an AssignmentID in status", func() {
		AfterEach(func() { cleanupResource() })

		It("should remove the finalizer without calling Delete", func() {
			resource := newResource(true)
			Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())

			deleteCalled := false
			svc := &fakePolicyAssignmentService{
				deleteFn: func(_ context.Context, _, _ string, _ []governancev1alpha1.AssignmentExemptionStatus, _ *governancev1alpha1.AssignmentIdentity) error {
					deleteCalled = true
					return nil
				},
			}

			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{NamespacedName: namespacedName})
			Expect(err).NotTo(HaveOccurred())
			Expect(deleteCalled).To(BeFalse())

			updated := &governancev1alpha1.AzurePolicyAssignment{}
			Expect(k8serrors.IsNotFound(k8sClient.Get(ctx, namespacedName, updated))).To(BeTrue())
		})
	})

	Context("When the resource does not exist", func() {
		It("should return no error", func() {
			svc := &fakePolicyAssignmentService{}
			_, err := newReconciler(svc).Reconcile(ctx, reconcile.Request{
				NamespacedName: types.NamespacedName{Name: "non-existent", Namespace: "default"},
			})
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
