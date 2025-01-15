package controller

import (
	"context"
	"sort"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	// fake client
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

func TestApplyTaintToNodePool(t *testing.T) {
	ctx := context.Background()

	// ★ 1. テスト用 Scheme を作成し、必要な型を登録
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	// 他に必要な API があれば続けて AddToScheme(...)

	// 2. テスト用の Node オブジェクトを作成
	testNode := &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-node",
			Labels: map[string]string{
				"beta.kubernetes.io/instance-type": "e2-medium",
			},
		},
	}

	// 3. FakeClient を用意し、Node を事前登録
	fakeClient := fake.NewClientBuilder().
		WithScheme(scheme). // ★ ここで同じ scheme を渡す
		WithObjects(testNode).
		Build()

	// 4. テスト対象となる Reconciler を作成 (FakeClient を差し込む)
	r := &MyComputeClassReconciler{
		Client: fakeClient,
		Scheme: scheme, // ★ Reconciler にも同じ scheme を渡す
	}

	// --- ここから下はそのまま ---
	nodeKey := client.ObjectKey{Name: "test-node"}
	nodeBefore := &corev1.Node{}
	require.NoError(t, fakeClient.Get(ctx, nodeKey, nodeBefore))
	require.Empty(t, nodeBefore.Spec.Taints)

	require.NoError(t, r.applyTaintToNodePool(ctx, "e2-medium"))

	nodeAfter := &corev1.Node{}
	require.NoError(t, fakeClient.Get(ctx, nodeKey, nodeAfter))
	require.Len(t, nodeAfter.Spec.Taints, 1)

	taint := nodeAfter.Spec.Taints[0]
	require.Equal(t, "my-compute-class", taint.Key)
	require.Equal(t, "e2", taint.Value)
	require.Equal(t, corev1.TaintEffectNoSchedule, taint.Effect)

	// 重複しないことをテスト
	require.NoError(t, r.applyTaintToNodePool(ctx, "e2-medium"))

	nodeAfterSecond := &corev1.Node{}
	require.NoError(t, fakeClient.Get(ctx, nodeKey, nodeAfterSecond))
	require.Len(t, nodeAfterSecond.Spec.Taints, 1)
}
