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
	"fmt"
	"os"
	"path/filepath"
	"testing"

	testingv1 "github.com/Pragmatics/orcatester/api/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestMain(m *testing.M) {
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases"), filepath.Join("..", "testbin", "crd")},
		ErrorIfCRDPathMissing: true,
	}

	//manually set the kubebuilder assets env variable to allow for easier testing within IDE
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		os.Setenv("KUBEBUILDER_ASSETS", filepath.Join("..", "testbin", "bin"))
	}
	cfg, err := testEnv.Start()
	defer testEnv.Stop()
	exitIfErrorOccurred(err)

	err = testingv1.AddToScheme(scheme.Scheme)
	exitIfErrorOccurred(err)

	var schemaGroupVersion = schema.GroupVersion{
		Group:   "tekton.dev",
		Version: "v1beta1",
	}
	scheme.Scheme.AddKnownTypes(schemaGroupVersion, &v1beta1.PipelineRun{}, &v1beta1.PipelineRunList{})
	metav1.AddToGroupVersion(scheme.Scheme, schemaGroupVersion)

	//+kubebuilder:scaffold:scheme

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	exitIfErrorOccurred(err)

	var tektonNamespace = corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tekton-pipelines",
		},
	}

	err = k8sClient.Create(context.Background(), &tektonNamespace)
	exitIfErrorOccurred(err)

	k8sManager, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	exitIfErrorOccurred(err)

	err = (&SuiteReconciler{
		Client: k8sManager.GetClient(),
		Scheme: k8sManager.GetScheme(),
	}).SetupWithManager(k8sManager)
	exitIfErrorOccurred(err)

	go func() {
		err = k8sManager.Start(ctrl.SetupSignalHandler())
		exitIfErrorOccurred(err)
	}()

	os.Exit(m.Run())
}

func exitIfErrorOccurred(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
