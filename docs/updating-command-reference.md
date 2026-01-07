
### Updating the kubeadm command reference documentation

kubeadm uses [Cobra](https://github.com/spf13/cobra) as a CLI library and the command line reference
documentation is generated automatically. The generated output can be found here:
* https://kubernetes.io/docs/reference/setup-tools/kubeadm/kubeadm/

If a command or a command flag is added or removed this has to be reflected in the documentation
on a new release. Some manual steps are still required.

#### Example scenario: adding a new sub-command

- Implemented a new kubeadm configuration sub-command called `kubeadm config newcommand` by
sending a PR for `kubernetes/kubernetes`.
- Run `./hack/update-generated-docs.sh`. This will generate files in the folder `./docs/admin/`.
Some of them will be for the new command - `*newcommand*`.
- In your local copy of `kubernetes/website` navigate to this folder:
`./content/en/docs/reference/setup-tools/kubeadm/generated`
- Copy the `*newcommand*` files from `kubernetes/docs/admin/` to the folder.
- Create a PR for `kubernetes/website` and add these files to your commit.
- Depending on the parent command of `newcommand` (in this case `config`) import the generated
sub-command in the parent command MD file like so:
	```
	## kubeadm config view {#cmd-config-newcommand}
	{{< include "generated/kubeadm_config_newcommand.md" >}}
	```
	Full example:
	* https://git.k8s.io/website/content/en/docs/reference/setup-tools/kubeadm/kubeadm-config.md
- Please note that these files will act only as placeholders with respect to the `kubernetes/website`
and they will later be overwritten with generated files by a [separate tool](https://github.com/kubernetes-sigs/reference-docs)
that also supports HTML styles. This process is managed by SIG Docs on each release.

#### Example scenario: removing a sub-command

- Remove the sub-command `kubeadm config newcommand` by sending a PR for `kubernetes/kubernetes`.
- When sending a PR for `kubernetes/website` make sure that you remove files related to `*newcommand*` in:
`./content/en/docs/reference/setup-tools/kubeadm/generated`
- Also, remove includes and any notes about this command in the parent command MD file.
- Make sure that you commit these changes in your PR for `kubernetes/website`.
