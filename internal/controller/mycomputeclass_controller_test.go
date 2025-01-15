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

// PrioritySortingTest では、MyComputeClassReconciler の HandlePriorityList が
// MyComputeClass.Spec.Properties を正しく Priority 順にソートするかを検証します。
var _ = Describe("MyComputeClass Priority Sorting", func() {
	const (
		testNamespace = "default"
		resourceName  = "test-mycomputeclass-priority"
	)

	ctx := context.Background()
	typeNamespacedName := types.NamespacedName{
		Namespace: testNamespace,
		Name:      resourceName,
	}

	BeforeEach(func() {
		// EnvTest 上に同名のリソースが残っていれば削除しておく
		mc := &scalingv1.MyComputeClass{}
		err := k8sClient.Get(ctx, typeNamespacedName, mc)
		if err == nil {
			_ = k8sClient.Delete(ctx, mc)
		} else if !k8serrors.IsNotFound(err) {
			// それ以外のエラーであればテストを中断
			Expect(err).NotTo(HaveOccurred())
		}

		// 新規 MyComputeClass リソースを作成
		// Priority がバラバラの順序で並んでいる
		testMC := &scalingv1.MyComputeClass{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: testNamespace,
			},
			Spec: scalingv1.MyComputeClassSpec{
				Properties: []scalingv1.InstanceProperty{
					{MachineFamily: "n2", Priority: 3},
					{MachineFamily: "e2", Priority: 2},
					{MachineFamily: "n1", Priority: 1},
				},
			},
		}
		err = k8sClient.Create(ctx, testMC)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		// 後片付けとして、テスト用 MyComputeClass を削除
		mc := &scalingv1.MyComputeClass{}
		err := k8sClient.Get(ctx, typeNamespacedName, mc)
		if err == nil {
			_ = k8sClient.Delete(ctx, mc)
		}
	})

	It("should return a Priority-sorted list via HandlePriorityList", func() {
		// Reconciler 作成 (GKEClient や他のフィールドは今回は使わない想定)
		r := &MyComputeClassReconciler{
			Client: k8sClient,
			Scheme: k8sClient.Scheme(),
		}

		// Reconcile Request を準備
		req := reconcile.Request{
			NamespacedName: typeNamespacedName,
		}

		// HandlePriorityList を直接呼んで Priority ソートされるかを確認
		sortedProps, err := r.HandlePriorityList(ctx, req)
		Expect(err).NotTo(HaveOccurred())
		Expect(sortedProps).To(HaveLen(3))

		// 返されたスライスが Priority の昇順であることを検証
		// (sortedProps[0].Priority <= sortedProps[1].Priority <= sortedProps[2].Priority)
		isSorted := sort.SliceIsSorted(sortedProps, func(i, j int) bool {
			return sortedProps[i].Priority < sortedProps[j].Priority
		})
		Expect(isSorted).To(BeTrue(), "Properties should be sorted by ascending Priority")

		// 具体的に Priority 値が 1,2,3 の順かも確認
		Expect(sortedProps[0].MachineFamily).To(Equal("n1"))
		Expect(sortedProps[1].MachineFamily).To(Equal("e2"))
		Expect(sortedProps[2].MachineFamily).To(Equal("n2"))
	})
})
