### kubeadm's release cycle and quarterly agenda

Kubernetes has a well-defined release process that improves with every release.
These notes provide some inspiration for how the Kubernetes developing flow actually can look like.

Most of these tips and tricks are general and not kubeadm specific.
But as we move more and more components out of the main Kubernetes repo, these communication
channels and escalation paths become increasingly important. Some of them are explained
in this doc.

Some of what's outlined below are just rough guidelines and processes that we use when developing kubeadm.
It makes sense to document this so the rest of the community can see what it looks like to be involved
in developing a core component, and/or to onboard new contributors to the process.

That said, nothing here is set in stone. See this as inspiration :)

#### First month of the quarter (coding)

- Create a new housekeeping tracking issue similar to [this one](https://github.com/kubernetes/kubeadm/issues/180).
- Remove e2e tests for k8s/kubeadm versions outside the support skew.
- Manage cherrypicks for the current stable branch
  - Make sure high-priority bug reports are triaged and fixed.
  - Create bug fixes, ensure they're merged, and cherry-pick them to the release branch.
  - Coordinate with the patch release team.
- Plan/prioritize features, testing work and bugs
  - At the beginning of each cycle the kubeadm team has a planning session during the kubeadm office hours meeting.
  - There people sign up for working on the high level features and objectives for the release.
  - Based on the planning session shift issues between milestones in the `kubernetes/kubeadm` repository.
- Go through _every_ open issue in the kubeadm repository.
  - As a general practice, every issue must be triaged **at least** once a quarter.
  - Close old, outdated issues.
- Write proposals for new and larger features
  - Often starting as a Google Doc that is easy to comment on/let others collaborate on.
  - Get consensus on the design during the kubeadm office hours meeting.
  - Convert the document to a KEP and send PR to `kubernetes/enhancements` repository. Assign KEP reviewers
  and ping people from other SIGs for feedback.
  - Update as needed by review comments, get LGTM and merge.
  - Make sure that a tracking issue for this feature exists in `kubernetes/enhancements` too. This is where the
  release team is going to observe the progress of this work and/or ask questions.
- Create new issues if needed
  - Break down the implementation of larger feature into smaller actionable issues, so that the work can be federated.
  - When possible, tag issues with the "help-wanted"/"good-first-issue" labels.
  - Use parent tracking issues to track/coordinate all the efforts.
- Start coding new features, fix bugs and increase unit testing coverage levels

#### Second month of the quarter (coding, docs and tests)

- Code, code, code :)
- Start writing docs and e2e tests for the new features.
- The kubeadm upgrade document needs a manual update each cycle:
https://kubernetes.io/docs/tasks/administer-cluster/kubeadm/kubeadm-upgrade/
- Go through the kubeadm related documents in the `kubernetes/website` and this repository and plan what has to be updated.
- Watch for new features in core Kubernetes that may affect kubeadm. Discuss what action has to be taken
by the kubeadm maintainers.
- Make sure that existing e2e test jobs are green at all times:
  https://testgrid.k8s.io/sig-cluster-lifecycle-kubeadm
  Without good signal there is no indication if new PRs are introducing regressions.

#### Third month of the quarter (stabilization)

- This month; code freeze is applied and no feature PRs may merge, only critical bug fixes
  - Right before code freeze you should ensure that all feature PRs for kubeadm are reviewed, LGTM'd and approved.
- Attend the Release Burndown Meetings whenever possible and/or communicate with the release team often to be on top of the release events.
- Marketing & announcements (optional)
  - Write a blog post on new features, possibly for inclusion in the "Five days of Kubernetes v1.X" blog post series on https://blog.kubernetes.io
  - Demonstrate new features at the Kubernetes Community Meeting.
- Make sure all critical bugfixes are prioritized and merged into the release.
- **IMPORTANT** Manual actions before a release:
  - Make sure that the default and minimum Kubernetes versions are up-to-date:
  https://github.com/kubernetes/kubernetes/blob/release-1.17/cmd/kubeadm/app/constants/constants.go#L412-L418
  - Make sure that the etcd versions in the same file are up-to-date:
  https://github.com/kubernetes/kubernetes/blob/release-1.17/cmd/kubeadm/app/constants/constants.go#L421-L427
  https://github.com/kubernetes/kubernetes/blob/release-1.17/cmd/kubeadm/app/constants/constants.go#L260-L264

### Other links

- A document on managing kubeadm e2e tests:
  https://git.k8s.io/kubeadm/docs/managing-e2e-tests.md
