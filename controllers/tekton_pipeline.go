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
	"encoding/hex"
	"math/rand"
	"strconv"
	"strings"

	v1 "github.com/Pragmatics/orcatester/api/v1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/util/json"
	"sigs.k8s.io/yaml"
)

const (
	pipelineRunKind       = "PipelineRun"
	pipelineRunApiVersion = "tekton.dev/v1beta1"
)

const (
	pipelineRunIdSize = 8
)

func randomId() string {
	b := make([]byte, pipelineRunIdSize/2)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func CreatePipelineRun(testNamespace string, suiteName string, testSpec *v1.TestSpec, previousAttempts int) v1beta1.PipelineRun {
	var jsonBytes, yamlToJsonErr = yaml.YAMLToJSON([]byte(testSpec.TektonPipelineRunYaml))
	if yamlToJsonErr != nil {
		suiteControllerLog.Error(yamlToJsonErr, "Failed to convert tekton pipeline run yaml to json.")
	}

	var pipelineRun v1beta1.PipelineRun
	if err := json.Unmarshal(jsonBytes, &pipelineRun); err != nil {
		suiteControllerLog.Error(err, "Failed to convert tekton pipeline run json to struct.")
	}

	pipelineRun.Namespace = testNamespace
	pipelineRun.Kind = pipelineRunKind
	pipelineRun.APIVersion = pipelineRunApiVersion
	if pipelineRun.Name == "" {
		pipelineRun.Name = strings.Join([]string{testSpec.Name, randomId()}, "-")
	}
	if testSpec.MaxRetries > 0 {
		pipelineRun.Name = strings.Join([]string{pipelineRun.Name, strconv.Itoa(previousAttempts + 1)}, "-")
	}
	if pipelineRun.Annotations == nil {
		pipelineRun.Annotations = map[string]string{}
	}
	pipelineRun.Annotations["testSuite"] = suiteName
	pipelineRun.Annotations["testName"] = testSpec.Name
	pipelineRun.Annotations["testAttempt"] = strconv.Itoa(previousAttempts + 1)
	if pipelineRun.Labels == nil {
		pipelineRun.Labels = map[string]string{}
	}
	pipelineRun.Labels["testSuite"] = suiteName
	return pipelineRun
}
