# Triaging and fixing kubeadm test failures

Kubeadm now has a series of end-to-end tests which run for every pull request
and merge action against the repo. Tests, however, often fail.

This document seeks to create a shared process that sets expectations for
community members around who should triage and fix kubeadm, or the tests
themselves, when they start failing.

## A new feature is submitted which inadvertently breaks correct tests

If tests begin to fail on a pull request which introduces new functionality or
refactors old, and the tests are still deemed relevant, it is the responsibility
of the contributor to fix the broken tests.

This process will tie into the usual feedback cycle for pull requests: if a PR
build fails, the CI process will inform the contributor of which tests are failing,
and the contributor can incorporate changes to get the e2e tests to pass again.

## A code fix is submitted that breaks old/incorrect tests

If tests fail for a submitted code fix, it is usually the case that the tests
themselves are either outdated or actually incorrect. The scope of the fix
should _also_ incorporate fixing test cases - whether unit or end-to-end.

Updating test cases should require detailed review from maintainers, who are
ultimately responsible for keeping this in check. For this reason, contributors
should be encouraged to work with and get the help of maintainers where
necessary.

## A ecosystem/upstream change happens that breaks current tests

If tests fail due to an upstream change to test-infra or a related ecosystem
project, it is the responsibility of maintainers to correct and recalibrate
tests.  

Ideally, maintainers and those accountable for such upstream projects should
make it a habit of routinely reaching out to relevant stakeholders when such
breaking changes occur. Test-infra maintainers could, for example, post to the
[sig-testing group](https://groups.google.com/forum/#!forum/kubernetes-sig-testing) with
upcoming changes, and maintainers of other projects can subscribe.

Without strict versioning, this contract between projects and upstream
will always be brittle - so regular communication is a must.

# Troubleshooting
## Create Cluster Fails
There is an issue in provisioning the cluster.

Test failure will look like:

```error during make -C /workspace/kubernetes-anywhere WAIT_FOR_KUBECONFIG=y deploy: exit status 2```

To debug further, dig into the collected logs
1. Determine if master was able to set up with kubeadm. Logs are under: ```/artifacts/master-node-name/serial-1.log```
2. Determine if nodes were able to set up with kubeadm. Logs are under: ```/artifacts/node-name/serial-1.log```
