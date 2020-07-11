package __performance

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	testutils "github.com/openshift-kni/performance-addon-operators/functests/utils"
	testclient "github.com/openshift-kni/performance-addon-operators/functests/utils/client"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/nodes"
	"github.com/openshift-kni/performance-addon-operators/functests/utils/pods"
	performancev1alpha1 "github.com/openshift-kni/performance-addon-operators/pkg/apis/performance/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("[performance]RT Kernel", func() {
	var testpod *corev1.Pod

	AfterEach(func() {
		if testpod == nil {
			return
		}
		if err := testclient.Client.Delete(context.TODO(), testpod); err == nil {
			pods.WaitForDeletion(testpod, 60*time.Second)
		}
	})

	It("[test_id:26861][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] should have RT kernel enabled", func() {

		profile := &performancev1alpha1.PerformanceProfile{}
		key := types.NamespacedName{
			Name: testutils.PerformanceProfileName,
		}
		err := testclient.Client.Get(context.TODO(), key, profile)
		Expect(err).ToNot(HaveOccurred(), "failed to get the performance profile")

		if profile.Spec.RealTimeKernel == nil || *profile.Spec.RealTimeKernel.Enabled == false {
			Skip("Skipping RT kernel test since profile does not have RT kernel set")
		}

		Eventually(func() string {

			// run uname -a in a busybox pod and get logs
			testpod = pods.GetTestPod()
			testpod.Namespace = testutils.NamespaceTesting
			testpod.Spec.Containers[0].Command = []string{"uname", "-a"}
			testpod.Spec.RestartPolicy = corev1.RestartPolicyNever
			testpod.Spec.NodeSelector = testutils.NodeSelectorLabels

			if err := testclient.Client.Create(context.TODO(), testpod); err != nil {
				return ""
			}

			if err := pods.WaitForPhase(testpod, corev1.PodSucceeded, 60*time.Second); err != nil {
				return ""
			}

			logs, err := pods.GetLogs(testclient.K8sClient, testpod)
			if err != nil {
				return ""
			}

			return logs

		}, 15*time.Minute, 30*time.Second).Should(ContainSubstring("PREEMPT RT"))

	})

	It("[test_id:28526][crit:high][vendor:cnf-qe@redhat.com][level:acceptance] Non worker-cnf node should not have RT kernel installed", func() {

		By("Skipping test if cluster does not have another available worker node")
		nonRTWorkerNodes, err := nodes.GetNonRTWorkers()
		Expect(err).ToNot(HaveOccurred())

		if len(nonRTWorkerNodes) == 0 {
			Skip("Skipping test because there are no additional non-cnf worker nodes")
		}

		cmd := []string{"uname", "-a"}
		kernel, err := nodes.ExecCommandOnNode(cmd, &nonRTWorkerNodes[0])
		Expect(err).ToNot(HaveOccurred(), "failed to execute uname")
		Expect(kernel).To(ContainSubstring("Linux"), "Node should have Linux string")
		Expect(kernel).NotTo(ContainSubstring("PREEMPT RT"), "Node should have non-RT kernel")
	})

})
