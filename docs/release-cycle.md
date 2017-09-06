### kubeadm's release cycle and quarterly agenda

Kubernetes has a well defined release process that improves with every release.
These notes are collected during the v1.8 release cycle, and provide some inspiration
for how the Kubernetes developing flow actually can look like.

Most of these tips and tricks are general and not kubeadm specific.
But as we move more and more components out of the main Kubernetes repo, these communication
channels and escalation paths become increasingly important. Some of them are explained
in this doc.

Some of what's outlined below are just rough guidelines and processes that we use when developing kubeadm.
It makes sense to document this so the rest of the community can see what it looks like to be involved
in developing a core component, and/or to onboard new contributors to the process.

That said, nothing here is set in stone. See this as inspiration :)

#### First month of the quarter (coding)

 - Manage cherrypicks for the current stable branch
   - Make sure high-priority bug reports are triaged and fixed.
   - Create bug fixes, ensure they're merged, and cherry pick them to the release branch.
   - Coordinate with the stable branch manager.
 - Plan/prioritize features, testing work and bugs
   - This is usually done in the SIG Cluster Lifecycle meetings, and a dedicated v1.X planning doc is made and referenced from the [meeting notes](https://docs.google.com/document/d/1deJYPIF4LmhGjDVaqrswErIrV7mtwJgovtLnPCDxP7U/edit#).
   - There people sign up for working on the high level features and objectives for the release
   - Create the right feature issues in `kubernetes/features` for what you want to accomplish for the quarter. Deadline is often the last day of the first month.
 - Go through _every_ open issue in the kubeadm repo.
   - As there are a small number of core maintainers working on kubeadm, there often isn't time to answer all feature requests or bugfixes, or to close old, outdated ones.
   - As a general practice, every issue must be triaged **at least** once a quarter.
 - Write proposals for new and larger features
   - Often starting as a Google Doc that is easy to comment on/let others collaborate on.
   - Get consensus on the design in the SIG
   - Convert to markdown, send PR to `kubernetes/community` and loop in the rest of the community (other relevant SIGs, if not looped in already)
   - Update as needed by review comments, get LGTM and merge.
 - Start coding new features, fix bugs and increase unit testing coverage levels
 - Ensure that the [`kubeadm-gce`](https://k8s-testgrid.appspot.com/sig-cluster-lifecycle#kubeadm-gce) is master blocking.
   - "Master blocking" means that this is the dashboard the release team looks at when they decide whether to cut an alpha.X release or not.
 - Cleanup work. As kubeadm supports deploying clusters one minor version below the CLI version, support for v1.7-specific workarounds may be dropped when the v1.9 cycle starts
   - A good example of this is Bootstrap Tokens and RBAC rules.
   - In v1.7, alpha Bootstrap Tokens were used. On upgrade to v1.8 they were slightly modified to use beta features and resource names
   - In v1.7, the `extensions/v1beta1` and `rbac.authorization.k8s.io/v1beta1` groups were used. In v1.8, `apps/v1beta2` and `rbac.authorization.k8s.io/v1` were added. Early in the v1.9 we can switch over to use `apps/v1beta2` and `rbac.authorization.k8s.io/v1` as those groups will always exist for the supported versions.
   - These are good examples of cleanup work that can be done. Also, `cmd/kubeadm/app/constants.MinimumControlPlaneVersion` should be bumped (in this example from `v1.7.0` to `v1.8.0`)
   - You might want to wait some time from the previous .0 release to make it easier to cherrypick possible bug fixes.

#### Second month of the quarter (coding)

 - Code, code, code :)
 - Create new e2e tests to cover functionality of kubeadm or that kubeadm depends on
   - Example: In the v1.8 cycle, e2e tests were added for the Node Authorizer and Bootstrap Tokens, both of them features kubeadm depend heavily on
   - Then those `[Feature:*]` e2e tests were enabled in the kubeadm e2e CI targeted at the `master` branch
 - Set up master upgrade tests from the current stable version to detect regressions quickly
 - Set up master kubeadm deploying current stable version of Kubernetes tests to detect such regressions quickly
 - Fill in release notes for the coming release in the draft (often in the `kubernetes/features` repo)
 - Open issues for everything that needs to be done for the upcoming release and add those to the related milestone.

#### Third month of the quarter (stabilization)

 - This month; code freeze is applied and no feature PRs may merge (if not an exception for the relevant feature is approved by the release team -- but this is unusual)
   - Right before code freeze you should ensure that all feature PRs for kubeadm are reviewed, LGTM'd and approved.
 - Create at least these three CI e2e jobs for the new release branch that is cut:
   - `kubeadm-gce-1.X`: Should run `kubeadm init` just normally as an user would, and is the main signal of whether kubeadm is working or not
   - `kubeadm-gce-1.{X-1}-on-1.X`: Should run `kubeadm init --kubernetes-version release-1.{X-1}` (one minor release less than kubeadm itself), as kubeadm supports that skew
   - `kubeadm-gce-upgrade-1.{X-1}-to-1.X`: Should run `kubeadm init --kubernetes-version release-1.{X-1}`, then `kubeadm upgrade apply {latest-1.X-release}`, then run e2e tests
   - Make sure that these all are in the `release-1.X-blocking` section of the testgrid.
     - This is really important, as this is the section/tab the release team is monitoring and basing go/no-go decisions on
 - Open a tracking issue in `kubernetes/kubernetes` where you tag the release team, add the v1.X milestone, and `priority/critical-urgent` and `status/approved-for-milestone` labels
   - This is needed so the release team has a clear escalation path for getting the latest information about the state of kubeadm, where release blockers, etc. can be discussed.
   - [Example](https://github.com/kubernetes/kubernetes/issues/51841)
   - Attend the Release Burndown Meetings whenever possible and/or communicate with the release team often to be on top of the release events.
 - Make sure all critical bugfixes get the v1.X milestone, `status/approved-for-milestone` label and merged in time.
 - Marketing & announcements
   - Write a blog post on new features, possibly for inclusion in the "Five days of Kubernetes v1.X" blog post series on https://blog.kubernetes.io
   - Demonstrate new features at the [Kubernetes Community Meeting](https://docs.google.com/document/d/1VQDIAB0OqiSjIHI8AWMvSdceWhnz56jNpZrLs6o7NJY/edit)
     and do the SIG Update on behalf of SIG Cluster Lifecycle
   - Sync with Developer Advocates and collaborate on writing about kubeadm features, also those who aren't the most highlighted one, but still useful
   - Write v1.X user-facing documentation in the `kubernetes/kubernetes.github.io` repo
 - **IMPORTANT**: Right before `v1.X.0-rc.1` is cut, bump the default Kubernetes version to use in kubeadm, like [this](https://github.com/kubernetes/kubernetes/pull/47440)
 - Until pushing of debs/rpms is an automated process, ping a Googler to push newly generated debs and rpms to the respective repos, can be verified via these links:
   - debs: https://packages.cloud.google.com/apt/dists/kubernetes-xenial/main/binary-amd64/Packages
   - rpms: https://packages.cloud.google.com/yum/repos/kubernetes-el7-x86_64/repodata/primary.xml
