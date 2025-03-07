// SPDX-License-Identifier: Apache-2.0
// Copyright 2022 Authors of KubeArmor

package visibility

import (
	"fmt"
	"time"

	"github.com/kubearmor/KubeArmor/tests/util"
	. "github.com/kubearmor/KubeArmor/tests/util"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = BeforeSuite(func() {
	// install wordpress-mysql app
	err := K8sApply([]string{"../smoke/res/wordpress-mysql-deployment.yaml"})
	Expect(err).To(BeNil())

	// delete all KSPs
	err = DeleteAllKsp()
	Expect(err).To(BeNil())

	// enable kubearmor port forwarding
	err = KubearmorPortForward()
	Expect(err).To(BeNil())
})

var _ = AfterSuite(func() {
	KubearmorPortForwardStop()
})

func getWpsqlPod(name string, ant string) string {
	pods, err := K8sGetPods(name, "wordpress-mysql", []string{ant}, 60)
	Expect(err).To(BeNil())
	Expect(len(pods)).To(Equal(1))
	return pods[0]
}

func isOperationsExpected(operations []string, expected []string) bool {
	if len(operations) != len(expected) {
		return true
	}
	for _, operation := range operations {
		found := false
		for _, expectedOp := range expected {
			if operation == expectedOp {
				found = true
				break
			}
		}
		if !found {
			fmt.Printf("Operation not found %v %v", operation, expected)
			return false
		}
	}
	return true
}

var _ = Describe("Visibility", func() {
	var wp string

	BeforeEach(func() {
		wp = getWpsqlPod("wordpress-",
			"container.apparmor.security.beta.kubernetes.io/wordpress: localhost/kubearmor-wordpress-mysql-wordpress-wordpress")
	})

	AfterEach(func() {
		KarmorLogStop()
		err := DeleteAllKsp()
		Expect(err).To(BeNil())
		// wait for policy deletion
		time.Sleep(5 * time.Second)
	})

	Describe("Visibility tests", func() {
		It("can get full visibility", func() {
			err := util.AnnotateNS("wordpress-mysql", "kubearmor-visibility", "file,process,network,capabilities")
			Expect(err).To(BeNil())

			// Start Kubearmor Logs
			err = KarmorLogStart("all", "wordpress-mysql", "", wp)
			Expect(err).To(BeNil())

			sout, _, err := K8sExecInPod(wp, "wordpress-mysql", []string{"bash", "-c", "ping google.com -c1"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)

			// check audit logs
			logs, _, err := KarmorGetLogs(5*time.Second, 50)
			Expect(err).To(BeNil())
			Expect(len(logs)).NotTo(Equal(0))

			expected := []string{
				"file",
				"process",
				"syscall",
				"network",
			}
			operations := GetOperations(logs)

			Expect(isOperationsExpected(operations, expected)).To(BeTrue())

		})

		It("can get only network visibility", func() {
			err := util.AnnotateNS("wordpress-mysql", "kubearmor-visibility", "network")
			Expect(err).To(BeNil())

			// Start Kubearmor Logs
			err = KarmorLogStart("all", "wordpress-mysql", "", wp)
			Expect(err).To(BeNil())

			sout, _, err := K8sExecInPod(wp, "wordpress-mysql", []string{"bash", "-c", "ping google.com -c1"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)

			// check audit logs
			logs, _, err := KarmorGetLogs(5*time.Second, 50)
			Expect(err).To(BeNil())
			Expect(len(logs)).NotTo(Equal(0))

			expected := []string{
				"network",
			}
			operations := GetOperations(logs)

			Expect(isOperationsExpected(operations, expected)).To(BeTrue())

		})

		It("can drop all visibility data", func() {
			err := util.AnnotateNS("wordpress-mysql", "kubearmor-visibility", "none")
			Expect(err).To(BeNil())

			// Start Kubearmor Logs
			err = KarmorLogStart("all", "wordpress-mysql", "", wp)
			Expect(err).To(BeNil())

			sout, _, err := K8sExecInPod(wp, "wordpress-mysql", []string{"bash", "-c", "ping google.com -c1"})
			Expect(err).To(BeNil())
			fmt.Printf("---START---\n%s---END---\n", sout)

			// check audit logs
			logs, _, err := KarmorGetLogs(5*time.Second, 50)
			Expect(err).To(BeNil())
			Expect(len(logs)).To(Equal(0))

		})

	})

})
