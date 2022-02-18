/*
Copyright 2021 Pragmatics.

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

package controllers

import (
	"context"
	"strconv"
	"strings"
	"testing"
	"time"

	testingv1 "github.com/Pragmatics/orcatester/api/v1"
	"github.com/stretchr/testify/assert"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// total wait time = waitIntervals * waitIntervalPeriod
const waitIntervals = 16
const waitIntervalPeriod = 250 * time.Millisecond

func BenchmarkSuiteStatusOneHundredTests(b *testing.B) {
	var testCount = 100
	var tests = make([]testingv1.TestSpec, testCount)
	var prs = make([]v1beta1.PipelineRun, testCount)
	var suiteName = "test-suite-performance"
	for i := 0; i < testCount; i++ {
		tests[i] = testingv1.TestSpec{
			Name:                  "test" + strconv.Itoa(i),
			TektonPipelineRunYaml: pipelineRunA,
		}
		prs[i] = CreatePipelineRun("pipelines", suiteName, &tests[i], 0)
		prs[i].Status.Status.Conditions = []apis.Condition{{
			Type:   apis.ConditionSucceeded,
			Status: v1.ConditionTrue,
		}}
	}
	var prList = v1beta1.PipelineRunList{}
	prList.Items = prs

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      suiteName,
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Tests: tests,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		currentSuiteStatus(&suite, &prList)
	}
}

func TestControllerCreatesPipelineRunsInOrderSuccess(t *testing.T) {
	var err error

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-suite1",
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Tests: []testingv1.TestSpec{{
				Name:                  "testa",
				TektonPipelineRunYaml: pipelineRunA,
			}, {
				Name:                  "testb",
				TektonPipelineRunYaml: pipelineRunB,
			}, {
				Name:                  "testc",
				TektonPipelineRunYaml: pipelineRunC,
			}},
		},
	}
	err = k8sClient.Create(context.Background(), &suite)
	failNowIfErrorPresent(t, err)

	var pipelineRun v1beta1.PipelineRun

	// first test is completed
	assertTestCount(t, "test-suite1", 1)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testa", "")
	updatePipelineStatus(t, v1.ConditionUnknown, &pipelineRun)
	assertSuiteStatus(t, "test-suite1", "RUNNING")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite1", "RUNNING")

	// second test is completed
	assertTestCount(t, "test-suite1", 2)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testb", "")
	updatePipelineStatus(t, v1.ConditionUnknown, &pipelineRun)
	assertSuiteStatus(t, "test-suite1", "RUNNING")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite1", "RUNNING")

	// third test is completed
	assertTestCount(t, "test-suite1", 3)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testc", "")
	updatePipelineStatus(t, v1.ConditionUnknown, &pipelineRun)
	assertSuiteStatus(t, "test-suite1", "RUNNING")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite1", "SUCCESS")

}

func TestControllerMarksSuiteAsFailedOnOneTestFailure(t *testing.T) {
	var err error

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-suite2",
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Tests: []testingv1.TestSpec{{
				Name: "testa",
			}, {
				Name: "testb",
			}, {
				Name: "testc",
			}},
		},
	}
	err = k8sClient.Create(context.Background(), &suite)
	failNowIfErrorPresent(t, err)

	var pipelineRun v1beta1.PipelineRun

	// first test is completed
	assertTestCount(t, "test-suite2", 1)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testa", "")
	updatePipelineStatus(t, v1.ConditionUnknown, &pipelineRun)
	assertSuiteStatus(t, "test-suite2", "RUNNING")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite2", "RUNNING")

	// second test is completed
	assertTestCount(t, "test-suite2", 2)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testb", "")
	updatePipelineStatus(t, v1.ConditionUnknown, &pipelineRun)
	assertSuiteStatus(t, "test-suite2", "RUNNING")
	updatePipelineStatus(t, v1.ConditionFalse, &pipelineRun)
	assertSuiteStatus(t, "test-suite2", "RUNNING")

	// third test is completed
	assertTestCount(t, "test-suite2", 3)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testc", "")
	updatePipelineStatus(t, v1.ConditionUnknown, &pipelineRun)
	assertSuiteStatus(t, "test-suite2", "RUNNING")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite2", "FAILURE")
}

func TestControllerCreatesPipelineRunsInParallelSuccess(t *testing.T) {
	var err error

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-suite3",
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Capacity: 2,
			Tests: []testingv1.TestSpec{{
				Name:                  "testa",
				TektonPipelineRunYaml: pipelineRunA,
			}, {
				Name:                  "testb",
				TektonPipelineRunYaml: pipelineRunB,
			}, {
				Name:                  "testc",
				TektonPipelineRunYaml: pipelineRunC,
			}},
		},
	}
	err = k8sClient.Create(context.Background(), &suite)
	failNowIfErrorPresent(t, err)

	var pipelineRun v1beta1.PipelineRun

	assertTestCount(t, "test-suite3", 2)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testa", "")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite3", "RUNNING")
	assertTestCount(t, "test-suite3", 3)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testb", "")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite3", "RUNNING")
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testc", "")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite3", "SUCCESS")
}

func TestControllerCreatesPipelineRunsInParallelWithRetriesSuccess(t *testing.T) {
	var err error

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-suite4",
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Capacity: 2,
			Tests: []testingv1.TestSpec{{
				Name:                  "testa",
				TektonPipelineRunYaml: pipelineRunA,
				MaxRetries:            1,
			}, {
				Name:                  "testb",
				TektonPipelineRunYaml: pipelineRunB,
			}, {
				Name:                  "testc",
				TektonPipelineRunYaml: pipelineRunC,
			}},
		},
	}
	err = k8sClient.Create(context.Background(), &suite)
	failNowIfErrorPresent(t, err)

	var pipelineRun v1beta1.PipelineRun

	assertTestCount(t, "test-suite4", 2)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testa", "1")
	updatePipelineStatus(t, v1.ConditionFalse, &pipelineRun)
	assertSuiteStatus(t, "test-suite4", "RUNNING")
	assertTestCount(t, "test-suite4", 3)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testa", "2")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite4", "RUNNING")
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testc", "")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testb", "")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite4", "SUCCESS")
}

func TestControllerCreatesPipelineRunsInParallelWithRetriesAndIsolationTestSuccess(t *testing.T) {
	var err error

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-suite5",
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Capacity: 2,
			Tests: []testingv1.TestSpec{{
				Name:                  "testa",
				TektonPipelineRunYaml: pipelineRunA,
				MaxRetries:            1,
			}, {
				Name:                  "testb",
				TektonPipelineRunYaml: pipelineRunB,
			}, {
				Name:                  "testc",
				TektonPipelineRunYaml: pipelineRunC,
				RunInIsolation:        true,
			}},
		},
	}
	err = k8sClient.Create(context.Background(), &suite)
	failNowIfErrorPresent(t, err)

	var pipelineRun v1beta1.PipelineRun

	assertTestCount(t, "test-suite5", 2)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testa", "1")
	updatePipelineStatus(t, v1.ConditionFalse, &pipelineRun)
	assertSuiteStatus(t, "test-suite5", "RUNNING")
	assertTestCount(t, "test-suite5", 3)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testa", "2")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite5", "RUNNING")
	assertTestCount(t, "test-suite5", 3)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testb", "")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite5", "RUNNING")
	assertTestCount(t, "test-suite5", 4)
	pipelineRun = findPipelineRunByPrefixAndSuffix(t, suite.Name, "testc", "")
	updatePipelineStatus(t, v1.ConditionTrue, &pipelineRun)
	assertSuiteStatus(t, "test-suite5", "SUCCESS")
}

func TestControllerInvalidSuiteNoTestsError(t *testing.T) {
	var err error

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-suite6",
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Capacity: 2,
			Tests:    []testingv1.TestSpec{},
		},
	}
	err = k8sClient.Create(context.Background(), &suite)
	failNowIfErrorPresent(t, err)
	assertSuiteStatus(t, "test-suite6", "FAILURE")
}

func TestControllerInvalidTestNameLengthError(t *testing.T) {
	var err error

	var baseTestName = "testa"
	var testName = baseTestName
	for i := 0; i < 52-len(baseTestName)+1; i++ {
		testName = testName + "a"
	}

	var suite = testingv1.Suite{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Suite",
			APIVersion: "testing.tsgpragc.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-suite7",
			Namespace: "tekton-pipelines",
		},
		Spec: testingv1.SuiteSpec{
			Capacity: 2,
			Tests: []testingv1.TestSpec{{
				Name:                  testName,
				TektonPipelineRunYaml: pipelineRunA,
				MaxRetries:            1,
			}},
		},
	}
	err = k8sClient.Create(context.Background(), &suite)
	failNowIfErrorPresent(t, err)
	assertSuiteStatus(t, "test-suite7", "FAILURE")
}

type PipelineRunTestYaml struct {
	PipelineRun string `yaml:"pipelineRun"`
}

func updatePipelineStatus(t *testing.T, status v1.ConditionStatus, run *v1beta1.PipelineRun) {
	run.Status.Conditions = []apis.Condition{{
		Type:   "Succeeded",
		Status: status,
		LastTransitionTime: apis.VolatileTime{
			Inner: metav1.Time{
				Time: time.Now(),
			},
		},
	}}
	err := k8sClient.Status().Update(context.Background(), run)
	failNowIfErrorPresent(t, err)
}

func findPipelineRunByPrefixAndSuffix(t *testing.T, suiteName string, prefix string, suffix string) v1beta1.PipelineRun {
	var runs v1beta1.PipelineRunList
	var result v1beta1.PipelineRun
	var failMessage = "Could not find pipelinerun with prefix: " + prefix + "and suffix: " + suffix
	eventuallyTrue(t, waitIntervals, waitIntervalPeriod, failMessage, func() bool {
		err := k8sClient.List(context.Background(), &runs, client.MatchingLabels{"testSuite": suiteName})
		if err != nil {
			return false
		}
		for i, pr := range runs.Items {
			if strings.HasPrefix(pr.Name, prefix) && strings.HasSuffix(pr.Name, suffix) {
				result = runs.Items[i]
				return true
			}
		}
		return false
	})
	return result
}

func assertTestCount(t *testing.T, suiteName string, count int) {
	var runs v1beta1.PipelineRunList
	var failMessage = suiteName + " never had a test count of: " + strconv.Itoa(count)
	eventuallyTrue(t, waitIntervals, waitIntervalPeriod, failMessage, func() bool {
		err := k8sClient.List(context.Background(), &runs, client.MatchingLabels{"testSuite": suiteName})
		if err != nil {
			return false
		}
		return len(runs.Items) == count
	})
}

func assertSuiteStatus(t *testing.T, suiteName string, suiteStatus string) {
	var suite testingv1.Suite
	var failMessage = suiteName + " never updated to status: " + suiteStatus
	eventuallyTrue(t, waitIntervals, waitIntervalPeriod, failMessage, func() bool {
		err := k8sClient.Get(context.Background(), client.ObjectKey{
			Namespace: "tekton-pipelines",
			Name:      suiteName,
		}, &suite)
		if err != nil {
			return false
		}
		return suite.Status.Status == suiteStatus
	})
}

func eventuallyTrue(t *testing.T, iterations int, interval time.Duration, failMessage string, f func() bool) {
	for i := 0; i < iterations; i++ {
		if !f() {
			time.Sleep(interval)
		} else {
			return
		}
	}
	assert.FailNow(t, failMessage)
}

func failNowIfErrorPresent(t *testing.T, err error) {
	if err != nil {
		assert.FailNow(t, err.Error())
	}
}

const pipelineRunA = `
spec:
serviceAccountName: build-bot
pipelineRef:
  name: test-a
workspaces:
- name: source
  volumeClaimTemplate:
	metadata:
	  name: temporary
	spec:
	  accessModes:
	  - ReadWriteOnce
	  resources:
		requests:
		  storage: 2Gi
podTemplate:
  securityContext:
	runAsUser: 65532
	runAsGroup: 2000
	fsGroup: 2000
params:
- name: imageName
  value: 'testparam'
`

const pipelineRunB = `
spec:
  serviceAccountName: build-bot
  pipelineRef:
    name: test-b
  workspaces:
  - name: source
    volumeClaimTemplate:
      metadata:
        name: temporary
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 2Gi
  podTemplate:
    securityContext:
      runAsUser: 65532
      runAsGroup: 2000
      fsGroup: 2000
  params:
  - name: imageName
    value: 'testparam'
`

const pipelineRunC = `
spec:
  serviceAccountName: build-bot
  pipelineRef:
    name: test-c
  workspaces:
  - name: source
    volumeClaimTemplate:
      metadata:
        name: temporary
      spec:
        accessModes:
        - ReadWriteOnce
        resources:
          requests:
            storage: 2Gi
  podTemplate:
    securityContext:
      runAsUser: 65532
      runAsGroup: 2000
      fsGroup: 2000
  params:
  - name: imageName
    value: 'testparam'
`
