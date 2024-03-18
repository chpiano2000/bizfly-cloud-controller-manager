package e2e_test

import (
	"e2e_test/test/framework"
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/types"
	core "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
)

func EnsuredService(isDeleted bool) types.GomegaMatcher {
	var ensureState string
	if isDeleted {
		ensureState = "DeletedLoadBalancer"
	} else {
		ensureState = "EnsuredLoadBalancer"
	}
	return And(
		WithTransform(func(e watch.Event) (string, error) {
			event, ok := e.Object.(*core.Event)
			if !ok {
				return "", fmt.Errorf("failed to poll event")
			}
			fmt.Println(event.Reason)
			return event.Reason, nil
		}, Equal(ensureState)),
	)
}

var _ = Describe("CCM E2E Tests", func() {
	var (
		err     error
		f       *framework.Invocation
		workers []string
	)

	const (
		bizflyProxyProtocol = "kubernetes.bizflycloud.vn/enable-proxy-protocol"
		bizflyNetworkType   = "kubernetes.bizflycloud.vn/load-balancer-network-type"
		bizflyNodeLabel     = "kubernetes.bizflycloud.vn/target-node-labels"
	)

	BeforeEach(func() {
		f = root.Invoke()
		workers, err = f.GetNodeList()
		Expect(err).NotTo(HaveOccurred())
		Expect(len(workers)).Should(BeNumerically(">=", 2))
	})

	ensureServiceLoadBalancer := func(namespace string, isDeleted bool) {
		watcher, err := f.LoadBalancer.GetServiceWatcher(namespace)
		Expect(err).NotTo(HaveOccurred())
		Eventually(watcher.ResultChan()).Should(Receive(EnsuredService(isDeleted)))
	}

	deployIngressController := func(namespace string) {
		Expect(f.LoadBalancer.DeployIngressController()).NotTo(HaveOccurred())
		ensureServiceLoadBalancer(namespace, false)
	}

	destroyIngressController := func() {
		Expect(f.LoadBalancer.UninstallIngressController()).NotTo(HaveOccurred())
	}

	Describe("Test", func() {
		Context("Create", func() {
			AfterEach(func() {
				err := root.Recycle()
				Expect(err).NotTo(HaveOccurred())
			})
			Context("Load Balancer with ingress hostname", func() {
				namespace := "ingress-nginx"

				BeforeEach(func() {
					namespace = "ingress-nginx"
					By("Deploying Nginx ingress controller")
					deployIngressController(namespace)
				})

				AfterEach(func() {
					By("Destroying Nginx ingress controller")
					destroyIngressController()
				})

				It("Should have external IP as a hostname", func() {
					var hostname []string
					By("Getting hostname")
					Eventually(func() error {
						hostname, err = f.LoadBalancer.GetLoadBalancerHostName(namespace, "ingress-nginx-controller")
						return err
					}).Should(BeNil())
					By("Checking for .nip.io domain")
					Eventually(hostname[0]).Should(ContainSubstring(".nip.io"))
				})
			})
		})
	})
})
