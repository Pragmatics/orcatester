# orcatester

Kubernetes operator that executes tests by creating Tekton pipelineruns.

![Alt text](docs/Orcatester.jpg?raw=true "Orcatester Diagram")

## How It Works

Define a kubernetes object representing a suite of tests.

```
apiVersion: testing.tsgpragc.com/v1
kind: Suite
metadata:
  name: service-tests
spec:
  capacity: 3 # defaults to 1 if unspecified
  tests:
    - name: "zap-test" # required
      maxRetries: 2 # defaults to 0 if unspecified
      runInIsolation: true # defaults to false if unspecified
      tektonPipelineRunYaml: |
        spec:
          serviceAccountName: build-bot
          pipelineRef:
            name: zap-test
          workspaces:
            - name: helm
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
            - name: "image_tag"
              value: "latest"
    - name: "cypress-test"
      maxRetries: 1
      tektonPipelineRunYaml: |
        spec:
          serviceAccountName: build-bot
          pipelineRef:
            name: cypress-test
          workspaces:
            - name: helm
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
            - name: "image_tag"
              value: "latest"
```

Once a suite is applied to your kubernetes cluster, the controller will spin up Tekton pipelines based on the supplied tektonPipelineRunYaml string.  The status of the suite will be updated as Tekton updates the pipelinerun status of each test tracked by the suite.

## Features

* Set maximum number of tests to run in parallel (capacity)
* Force individual tests to run in isolation (runInIsolation)
* Allow test retries (maxRetries)

## Restrictions

The recommended limit of tests per suite is 100.  Runtime
improvements may be planned in the future to allow for additional
tests per suite.

## Installing on to a Cluster

Note: It is recommend to check out the code that corresponds to the image you want to deploy.
The below example is for a release tagged 1.0.0

```
git clone https://github.com/Pragmatics/orcatester
git checkout 1.0.0
make install
make deploy IMG=TODO
```

## Contributing

### Downloading Go Deps

```
go mod tidy
```
or
```
go get .
```

### Running Tests

Run the below command at least once after cloning repo. This will download all of the
necessary dependencies.  A test apiserver is spun up on developer's machines that will be destroyed
after the tests are ran.
```
make test
```
Additional execution of tests can be ran using the below command.
```
go test ./...
```
