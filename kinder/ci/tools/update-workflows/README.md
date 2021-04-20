## update-workflows

Update-workflows is a tool that can be used to update kinder workflows
and test-infra Prow jobs.

### Example usage

```shell
go run ./cmd/main.go --config config.yaml --kubernetes-version v1.21 \
  --path-test-infra ./test-infra --path-workflows ./workflows \
  --image-test-infra v20210403-e49d2c6 --skew-size=3
```

### Flags

- `--config` must point to a static configuration file (see bellow).
- `--kubernetes-version` is the Kubernetes version to be tested. It is the base
of all the version skew in jobs. When a new k8s release branch is created the tool
must be run with the new release version.
- `--image-test-infra` is a portion of the tag used for running an kubeadm e2e test job
in Prow. Given the `kubernetes/test-infra` repository is cloned in `test-infra`, it can
be obtained by running the following command:

  ```shell
  grep image test-infra/config/jobs/kubernetes/sig-cluster-lifecycle -r | \
    head -1 | cut -d ':' -f 4 | cut -d '-' -f-2
  ```
- `--path-*` are paths where the files will be generated.
- `--skew-size` is the size of the k8s skew. If the value is `N`, the oldest k8s version
that tests will be generated for will be `kubernetes-version - N`.

### Config

See the inline comments to understand more about the `--config` format:

```yaml
# a job group is an object that contain multiple e2e tests for different version
jobGroups:
- name: foo # name of the job group
  # this means that the feature requires a minimum Kubernetes versions;
  # jobs for a Kubernetes version older than this value are not generated
  minimumKubernetesVersion: v1.19.0
  # contains settings for the test-infra job format
  testInfraJobSpec:
    # file name in test-infra to contain all jobs
    targetFile: kubeadm-kinder-foo.yaml
    # test-infra template
    template: ./templates/testinfra/kubeadm-kinder-foo.yaml
  kinderWorkflowSpec:
    # a file format to be use when writing kinder workflows
    targetFile: foo-{{ .KubernetesVersion }}.yaml
    # path to a kinder template
    template: ./templates/workflows/foo.yaml
    # additional files to copy to the workflow output directory
    additionalFiles:
    - ./templates/workflows/foo-tasks.yaml
  # a list of job objects containing version skew between k8s, kubelet and kubeadm version
  jobs:
  - kubernetesVersion: latest # 'latest' is interpreted as 'latest' version labels and main git branches
    kubeletVersion: -1 # 'kubernetes-version - 1'
    # skip versions 'kubernetes-version' and 'kubernetes-version - 1' (specific for kubelet skew tests)
    skipVersions: [ 0, -1 ]
  - kubernetesVersion: 0 # 'kubernetes-version'
    kubeletVersion: +1 # 'kubernetes-version + 1'
```

### Template functions and variables

#### Functions

- `dashVer`: takes a version formatted like `1.20` and converts it to `1-20`.
If the input is `latest` returns it untouched. Used for test-infra job names
that cannot have `.`.
- `ciLabelFor`: takes a version like `1.20` and returns `latest-1.20`.
If the input is `latest` returns it untouched. Used for `ci/latest-foo` labels.
- `branchFor`: takes a version like `1.20` and returns `release-1.20`
If the input is `latest` returns the `master`. Used for test-infra branch clones.
- `imageVer`: takes a version like `1.20` and returns it untouched unless it
is `latest`, in which case it returns `master`. Used for test-infra images.
- `sigReleaseVer`: same as `imageVer`. Used for SIG Release test-grid dashboards.

#### Variables

- `KubernetesVersion`: replaced with `kubernetesVersion` from job objects.
- `KubeletVersion`: replaced with `kubernetesVersion` from job objects.
- `KubeadmVersion`: replaced with `kubeadmVersion` from job objects.
- `InitVersion`: replaced with `initVersion` from job objects. This is the Kubernetes version
before upgrade.
- `TargetFile`: replace with `kinderWorkflowSpec.targetFile` or `testInfraJobSpec.targetFile`.
- `SkipVersions`: replaced with `skipVersions` from job objects. Used in kubelet skew workflows
to pass a list of versions to skip in a `ginkgoSkip` variable.
- `TestInfraImage`: replaced with the `--image-test-infra`.
- `JobInterval`: replaced with a static value depending on the version the job is testing.
- `AlertAnnotations`: replaced with two lines containing `testgrid-num-failures-to-alert` and
`testgrid-alert-stale-results-hours` with static values that depend on the version that a job
is testing.
- `WorkflowFile`: replaced with `kinderWorkflowSpec.targetFile`.
