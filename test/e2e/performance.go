/*
Copyright 2019 The Kubernetes Authors.

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

package e2e

import (
	"fmt"
	"strings"

	"sigs.k8s.io/azuredisk-csi-driver/test/e2e/driver"
	"sigs.k8s.io/azuredisk-csi-driver/test/e2e/testsuites"

	"github.com/onsi/ginkgo"
	v1 "k8s.io/api/core/v1"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = ginkgo.Describe("[azuredisk-csi-e2e] Performance", func() {
	f := framework.NewDefaultFramework("azuredisk")

	var (
		cs         clientset.Interface
		ns         *v1.Namespace
		testDriver driver.PVTestDriver
	)

	ginkgo.BeforeEach(func() {
		cs = f.ClientSet
		ns = f.Namespace
		testDriver = driver.InitAzureDiskDriver()
	})

	testDriver = driver.InitAzureDiskDriver()
	ginkgo.It(fmt.Sprintf("should create a pod with 8 PVCs"), func() {
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver: testDriver,
			Pods:      []testsuites.PodDetails{generatePodWithMultipleVolumes(8)},
		}
		test.Run(cs, ns)
	})

	ginkgo.FIt(fmt.Sprintf("should create 8 pods with 1 PVC each and schedule them to the same node"), func() {
		test := testsuites.DynamicallyProvisionedCmdVolumeTest{
			CSIDriver:    testDriver,
			Pods:         generatePods(8),
			ColocatePods: true,
		}
		test.Run(cs, ns)
	})
})

func generatePodWithMultipleVolumes(numVolumes int) testsuites.PodDetails {
	var cmd strings.Builder
	var volumes []testsuites.VolumeDetails
	for i := 1; i <= numVolumes; i++ {
		if cmd.String() != "" {
			cmd.WriteString(" && ")
		}

		// Write 'hello world' to a file called data and save it on the mounted disk
		cmd.WriteString(fmt.Sprintf("echo 'hello world' > /mnt/test-%d/data && grep 'hello world' /mnt/test-%d/data", i, i))
		volumes = append(volumes, testsuites.VolumeDetails{
			ClaimSize: "10Gi",
			VolumeMount: testsuites.VolumeMountDetails{
				NameGenerate:      "test-volume-",
				MountPathGenerate: "/mnt/test-",
			},
		})
	}
	return testsuites.PodDetails{
		Cmd:     cmd.String(),
		Volumes: volumes,
	}
}

func generatePods(numPods int) []testsuites.PodDetails {
	var pods []testsuites.PodDetails
	for i := 0; i < numPods; i++ {
		pods = append(pods, testsuites.PodDetails{
			Cmd: "while true; do echo $(date -u) >> /mnt/test-1/data; sleep 1; done",
			Volumes: []testsuites.VolumeDetails{
				{
					ClaimSize: "10Gi",
					VolumeMount: testsuites.VolumeMountDetails{
						NameGenerate:      "test-volume-",
						MountPathGenerate: "/mnt/test-",
					},
				},
			},
		})
	}
	return pods
}
