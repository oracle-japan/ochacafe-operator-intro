package controllers

import (
	"context"
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	ochacafev1alpha1 "github.com/oracle-japan/ochacafe-operator-intro/api/v1alpha1"
)

var _ = Describe("OchaCafe Operator", func() {
	//! [setup]
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		err := k8sClient.DeleteAllOf(ctx, &ochacafev1alpha1.Ochacafe{}, client.InNamespace("defualt"))
		Expect(err).NotTo(HaveOccurred())
		err = k8sClient.DeleteAllOf(ctx, &appsv1.Deployment{}, client.InNamespace("defualt"))
		Expect(err).NotTo(HaveOccurred())

		mgr, err := ctrl.NewManager(cfg, ctrl.Options{
			Scheme: scheme,
		})
		Expect(err).ToNot(HaveOccurred())

		reconciler := OchacafeReconciler{
			Client: k8sClient,
			Scheme: scheme,
		}
		err = reconciler.SetupWithManager(mgr)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})
	//! [setup]

	//! [test]
	It("should create Deployment", func() {
		ochacafe_1 := newOcha()
		err := k8sClient.Create(ctx, ochacafe_1)
		time.Sleep(100 * time.Millisecond)
		Expect(err).NotTo(HaveOccurred())

		dep := appsv1.Deployment{}
		Eventually(func() error {
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "ochacafe-app"}, &dep)
		}).Should(Succeed())
		Expect(dep.Spec.Replicas).Should(Equal(pointer.Int32Ptr(3)))
		Expect(dep.Spec.Template.Spec.Containers[0].Image).Should(Equal("nginx"))
	})

	It("should update status", func() {
		ochacafe_info := ochacafev1alpha1.Ochacafe{}
		Eventually(func() error {
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "default", Name: "ochacafe-app"}, &ochacafe_info)
			if err != nil {
				return err
			}
			if ochacafe_info.Status.Nodes != nil {
				return errors.New("status should be updated")
			}
			return nil
		}).Should(Succeed())
	})
	//! [test]
})

func newOcha() *ochacafev1alpha1.Ochacafe {
	return &ochacafev1alpha1.Ochacafe{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ochacafe-app",
			Namespace: "default",
		},
		Spec: ochacafev1alpha1.OchacafeSpec{
			Size:  3,
			Image: "nginx",
		},
	}
}
