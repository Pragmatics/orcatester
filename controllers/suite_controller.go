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
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	testingv1 "github.com/Pragmatics/orcatester/api/v1"
)

// SuiteReconciler reconciles a Suite object
type SuiteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const updateObjectErrorMessage = "the object has been modified; please apply your changes to the latest version and try again"

var (
	suiteControllerLog = ctrl.Log.WithName("suite controller")
)

//+kubebuilder:rbac:groups=testing.tsgpragc.com,resources=suites,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=testing.tsgpragc.com,resources=suites/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=testing.tsgpragc.com,resources=suites/finalizers,verbs=update
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Suite object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *SuiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	var suite testingv1.Suite
	suiteControllerLog.Info(strings.Join([]string{"Getting suite: ", req.Name}, ""))
	if err := r.Get(ctx, req.NamespacedName, &suite); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !isValidSuite(&suite) {
		if err := r.Status().Update(ctx, &suite); err != nil {
			// https://github.com/kubernetes/kubernetes/issues/28149
			if strings.Contains(err.Error(), updateObjectErrorMessage) {
				suiteControllerLog.Info(strings.Join([]string{"Expected conflict occurred updating state. Requeuing (in 1 second) request for: ", suite.Name}, ""))
				return reconcile.Result{RequeueAfter: time.Second * 1}, nil
			}
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		return ctrl.Result{}, errors.New(strings.Join([]string{"Invalid suite: ", suite.Name}, ""))
	}

	var pipelineRunList v1beta1.PipelineRunList
	if err := r.List(ctx, &pipelineRunList, client.MatchingLabels{"testSuite": suite.Name}); err != nil {
		suiteControllerLog.Error(err, strings.Join([]string{"Unable to list pipeline runs for suite: ", suite.Name}, ""))
		return ctrl.Result{}, err
	}

	suiteControllerLog.Info(strings.Join([]string{"Getting updated status for: ", suite.Name}, ""))
	currentSuiteStatus(&suite, &pipelineRunList)
	r.startNextPipelineJob(ctx, &suite)
	suiteControllerLog.Info(strings.Join([]string{"Updating suite status for: ", suite.Name}, ""))
	if err := r.Status().Update(ctx, &suite); err != nil {
		// https://github.com/kubernetes/kubernetes/issues/28149
		if strings.Contains(err.Error(), updateObjectErrorMessage) {
			suiteControllerLog.Info(strings.Join([]string{"Expected conflict occurred updating state. Requeuing (in 1 second) request for: ", suite.Name}, ""))
			return reconcile.Result{RequeueAfter: time.Second * 1}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	suiteControllerLog.Info(strings.Join([]string{"Successfully updated suite status for: ", suite.Name}, ""))
	return ctrl.Result{}, nil
}

func (r *SuiteReconciler) startNextPipelineJob(ctx context.Context, suite *testingv1.Suite) {
	var currentlyRunning = currentJobCount(suite)
	var maxJobs = 1
	if suite.Spec.Capacity != 0 {
		maxJobs = suite.Spec.Capacity
	}

	if currentlyRunning >= maxJobs {
		return
	}

	if currentlyRunning == 0 {
		r.startNextPendingJob(ctx, suite, true)
	} else if !isIsolationTestRunning(suite) {
		r.startNextPendingJob(ctx, suite, false)
	}
	suiteControllerLog.Info("Waiting for tests to complete before starting new jobs.")
}

func currentJobCount(suite *testingv1.Suite) int {
	var currentlyRunning = 0
	for _, testStatus := range suite.Status.Tests {
		if testStatus.Status == "RUNNING" {
			currentlyRunning++
		}
	}
	return currentlyRunning
}

func (r *SuiteReconciler) startNextPendingJob(ctx context.Context, suite *testingv1.Suite, allowIsolation bool) {
	for _, testStatus := range suite.Status.Tests {
		var test = getTestByName(testStatus.Name, &suite.Spec.Tests)
		if testStatus.Status == "PENDING" && (!test.RunInIsolation || allowIsolation) {
			r.createPipelineJob(testStatus.Name, testStatus.Attempts, ctx, suite)
			return
		}
	}
}

func isIsolationTestRunning(suite *testingv1.Suite) bool {
	for _, testStatus := range suite.Status.Tests {
		var test = getTestByName(testStatus.Name, &suite.Spec.Tests)
		if testStatus.Name == test.Name && testStatus.Status == "RUNNING" && test.RunInIsolation {
			return true
		}
	}
	return false
}

func (r *SuiteReconciler) createPipelineJob(testName string, attempts int, ctx context.Context, suite *testingv1.Suite) {
	var test = getTestByName(testName, &suite.Spec.Tests)
	var pipelinerun = CreatePipelineRun(suite.Namespace, suite.Name, test, attempts)
	suiteControllerLog.Info(strings.Join([]string{"Starting pipeline run for test: ", test.Name}, " "))
	if err := r.Create(ctx, &pipelinerun); err != nil {
		suiteControllerLog.Error(err, "failed to create pipeline")
	}
}

func getTestByName(testName string, testSpecs *[]testingv1.TestSpec) *testingv1.TestSpec {
	for _, test := range *testSpecs {
		if test.Name == testName {
			return &test
		}
	}
	return nil
}

func currentSuiteStatus(suite *testingv1.Suite, pipelineRunList *v1beta1.PipelineRunList) string {
	suite.Status = testingv1.SuiteStatus{
		Status: "PENDING",
		Tests:  []testingv1.TestStatus{},
	}
	var statusMap = make(map[string]int)
	for _, test := range suite.Spec.Tests {
		var testStatus, attempts = currentTestStatus(&test, pipelineRunList)
		statusMap[testStatus] = statusMap[testStatus] + 1
		suite.Status.Tests = append(suite.Status.Tests, testingv1.TestStatus{
			Name:     test.Name,
			Status:   testStatus,
			Attempts: attempts,
		})
	}

	if statusMap["SUCCESS"] == len(suite.Spec.Tests) {
		suite.Status.Status = "SUCCESS"
	} else if statusMap["FAILURE"] > 0 && statusMap["SUCCESS"]+statusMap["FAILURE"] == len(suite.Spec.Tests) {
		suite.Status.Status = "FAILURE"
	} else {
		suite.Status.Status = "RUNNING"
	}
	return suite.Status.Status
}

func currentTestStatus(testSpec *testingv1.TestSpec, runs *v1beta1.PipelineRunList) (string, int) {
	var attempts = 0
	var statusMap = make(map[string]int)

	for _, pipelineRun := range runs.Items {
		if pipelineRun.Annotations["testName"] == testSpec.Name {
			var runStatus = pipelineRunStatus(&pipelineRun)
			statusMap[runStatus] = statusMap[runStatus] + 1
			attempts++
		}
	}

	if statusMap["SUCCESS"] > 0 {
		return "SUCCESS", attempts
	} else if statusMap["FAILURE"] > testSpec.MaxRetries {
		return "FAILURE", attempts
	} else if attempts > 0 && statusMap["FAILURE"] < attempts {
		return "RUNNING", attempts
	} else {
		return "PENDING", attempts
	}
}

func pipelineRunStatus(pipelineRun *v1beta1.PipelineRun) string {
	if len(pipelineRun.Status.Status.Conditions) > 0 {
		var condition = pipelineRun.Status.Status.Conditions[0]
		if condition.Type == "Succeeded" {
			if condition.IsTrue() {
				return "SUCCESS"
			} else if condition.IsFalse() {
				return "FAILURE"
			} else {
				return "RUNNING"
			}
		} else {
			return "RUNNING"
		}
	} else {
		return "RUNNING"
	}
}

const retriesCharLength = 3

func isValidSuite(suite *testingv1.Suite) bool {
	if len(suite.Spec.Tests) == 0 {
		suite.Status.Status = "FAILURE"
		suiteControllerLog.Info("")
		return false
	}
	var length = 63 - pipelineRunIdSize - retriesCharLength
	for _, test := range suite.Spec.Tests {
		if len(test.Name) > length {
			suiteControllerLog.Info(strings.Join([]string{"Invalid test name for suite: ", suite.Name, " test: ", test.Name}, ""))
			suiteControllerLog.Info(strings.Join([]string{"Tests have a max name length of: ", strconv.Itoa(length)}, ""))
			suite.Status.Status = "FAILURE"
			return false
		}
	}
	return true
}

// SetupWithManager sets up the controller with the Manager.
func (r *SuiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&testingv1.Suite{}).
		Watches(&source.Kind{Type: &v1beta1.PipelineRun{}}, handler.EnqueueRequestsFromMapFunc(mapFn)).
		Complete(r)
}

var mapFn handler.MapFunc = func(object client.Object) []reconcile.Request {
	var testSuiteName = object.GetAnnotations()["testSuite"]
	if len(testSuiteName) == 0 {
		return []reconcile.Request{}
	}
	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      testSuiteName,
				Namespace: object.GetNamespace(),
			},
		},
	}
}
