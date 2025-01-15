package controller

import (
	"context"
	"sort"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	scalingv1 "tryu.com/my-computeclass/api/v1"
)

var _ = Describe("MyComputeClass Priority Sorting Test", func() {
	const (
		testNamespace = "default"
		resourceName  = "test-mycomputeclass"
	)

	ctx := context.Background()
	typeNamespacedName := types.NamespacedName{
		Namespace: testNamespace,
		Name:      resourceName,
	}

	BeforeEach(func() {
		// cleanup mycomputeclass resource
		mc := &scalingv1.MyComputeClass{}
		err := k8sClient.Get(ctx, typeNamespacedName, mc)
		if err == nil {
			_ = k8sClient.Delete(ctx, mc)
		} else if !k8serrors.IsNotFound(err) {
			Expect(err).NotTo(HaveOccurred())
		}

		// create mycomputeclass resource with priorities unsorted
		testMC := &scalingv1.MyComputeClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: testNamespace,
			},
			Spec: scalingv1.MyComputeClassSpec{
				Properties: []scalingv1.InstanceProperty{
					{MachineFamily: "n2", Priority: 3},
					{MachineFamily: "e2", Priority: 2},
					{MachineFamily: "n2d", Priority: 1},
				},
			},
		}
		err = k8sClient.Create(ctx, testMC)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// delete mycomputeclass resource after each test
		mc := &scalingv1.MyComputeClass{}
		err := k8sClient.Get(ctx, typeNamespacedName, mc)
		if err == nil {
			_ = k8sClient.Delete(ctx, mc)
		}
	})

	It("should return a Priority-sorted list via HandlePriorityList", func() {
		// initialize MyComputeClassReconciler
		r := &MyComputeClassReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		req := reconcile.Request{
			NamespacedName: typeNamespacedName,
		}

		// call HandlePriorityList
		sortedProps, err := r.HandlePriorityList(ctx, req)
		Expect(err).NotTo(HaveOccurred())
		Expect(sortedProps).To(HaveLen(3))

		// check if the properties are sorted by Priority
		isSorted := sort.SliceIsSorted(sortedProps, func(i, j int) bool {
			return sortedProps[i].Priority < sortedProps[j].Priority
		})
		Expect(isSorted).To(BeTrue(), "Properties should be sorted by ascending Priority")

		// check if the properties are sorted by Priority
		Expect(sortedProps[0].MachineFamily).To(Equal("n2d"))
		Expect(sortedProps[1].MachineFamily).To(Equal("e2"))
		Expect(sortedProps[2].MachineFamily).To(Equal("n2"))
	})
})
