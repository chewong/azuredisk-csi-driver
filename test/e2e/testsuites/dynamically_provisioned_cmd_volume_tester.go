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

package testsuites

import (
	"math/rand"
	"strconv"
	"strings"
	"sync"

	"sigs.k8s.io/azuredisk-csi-driver/test/e2e/driver"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientset "k8s.io/client-go/kubernetes"
)

var (
	tpods    []*TestPod
	cleanups []func()
	mux      sync.Mutex
	wg       sync.WaitGroup
	nodeName string
)

// DynamicallyProvisionedCmdVolumeTest will provision required StorageClass(es), PVC(s) and Pod(s)
// Waiting for the PV provisioner to create a new PV
// Testing if the Pod(s) Cmd is run with a 0 exit code
type DynamicallyProvisionedCmdVolumeTest struct {
	CSIDriver    driver.DynamicPVTestDriver
	Pods         []PodDetails
	ColocatePods bool
}

func (t *DynamicallyProvisionedCmdVolumeTest) Run(client clientset.Interface, namespace *v1.Namespace) {
	// Generate tpods in parallel to decrease test duration
	for i := range t.Pods {
		wg.Add(1)
		go t.generateTestPod(i, client, namespace)
	}
	wg.Wait()

	defer func() {
		for _, c := range cleanups {
			wg.Add(1)
			go func(c func()) {
				c()
				wg.Done()
			}(c)
		}
		wg.Wait()
	}()

	// Select a random node to schedule all the pods to, if coption is true
	if t.ColocatePods {
		nodes, err := client.CoreV1().Nodes().List(metav1.ListOptions{})
		gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		var nodeNames []string
		for _, n := range nodes.Items {
			if strings.Contains(n.Name, "master") {
				continue
			}
			nodeNames = append(nodeNames, n.Name)
		}
		nodeName = nodeNames[rand.Int()%len(nodeNames)]
	}

	for i := range tpods {
		wg.Add(1)
		go t.deployTestPod(i, client, namespace)
	}
	wg.Wait()
}

func (t *DynamicallyProvisionedCmdVolumeTest) generateTestPod(i int, client clientset.Interface, namespace *v1.Namespace) {
	pod := t.Pods[i]
	tpod := NewTestPod(client, namespace, pod.Cmd)

	var pvcWG sync.WaitGroup
	for i, v := range pod.Volumes {
		pvcWG.Add(1)
		// Setup each PVC in parallel
		go func(vIndex string, v VolumeDetails) {
			tpvc, pvcCleanup := v.SetupDynamicPersistentVolumeClaim(client, namespace, t.CSIDriver)
			mux.Lock()
			defer mux.Unlock()
			tpod.SetupVolume(tpvc.persistentVolumeClaim, v.VolumeMount.NameGenerate+vIndex, v.VolumeMount.MountPathGenerate+vIndex, v.VolumeMount.ReadOnly)
			cleanups = append(cleanups, pvcCleanup...)
			pvcWG.Done()
		}(strconv.Itoa(i+1), v)
	}
	pvcWG.Wait()

	mux.Lock()
	defer mux.Unlock()
	tpods = append(tpods, tpod)
	wg.Done()
}

func (t *DynamicallyProvisionedCmdVolumeTest) deployTestPod(i int, client clientset.Interface, namespace *v1.Namespace) {
	tpod := tpods[i]
	if t.ColocatePods && nodeName != "" {
		tpod.SetNodeSelector(map[string]string{v1.LabelHostname: nodeName})
	}

	ginkgo.By("deploying the pod")
	tpod.Create()
	defer tpod.Cleanup()

	if t.ColocatePods {
		ginkgo.By("checking that the pod is running")
		tpod.WaitForRunning()
	} else {
		ginkgo.By("checking that the pod's command exits with no error")
		tpod.WaitForSuccess()
	}

	wg.Done()
}
