/*
Copyright 2019 The Kubernetes Authors.

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

/*
verify_manifest_lists.go

This program contains tests for Docker schema 2, multi-arch manifest lists
that kubeadm requires for every release.

First, it tries to get the latest Kubernetes release tags from GitHub.
The list of release tags is filter so that it doesn't contain versions
that are either too old or are known to not support manifest lists.

It then creates a map of required images and image tags by looking
at the constants defined in the kubeadm source code for a release
branch estimated from a release tag.

Once the images and image tags are defined, the program starts downloading
the manifest lists from GCR, but also the actual images and layers,
while verifying their contents and sizes. The layer download can be
skipped by toggling `downloadLayers` to false.

Results are cached so that a certain "image:tag" doesn't have to be verified
more than once.
*/

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/util/version"
)

const (
	// optional download of image layers.
	downloadLayers = false

	// default timeout for HTTP requests in seconds.
	defaultHTTPTimeout = 10

	// first release that is known to have full support.
	firstKnownVersion = "v1.12.0-rc.1"

	registryBucket = "https://registry.k8s.io/v2"

	typeManifestList = "application/vnd.docker.distribution.manifest.list.v2+json"
	typeManifest     = "application/vnd.docker.distribution.manifest.v2+json"
	typeLayer        = "application/vnd.docker.image.rootfs.diff.tar.gzip"

	messageStart = `
             _ ___                      _ ___         _      _ _     _
 _ _ ___ ___|_|  _|_ _    _____ ___ ___|_|  _|___ ___| |_   | |_|___| |_ ___
| | | -_|  _| |  _| | |  |     | .'|   | |  _| -_|_ -|  _|  | | |_ -|  _|_ -|
 \_/|___|_| |_|_| |_  |  |_|_|_|__,|_|_|_|_| |___|___|_|    |_|_|___|_| |___|
                  |___|

`

	messageSuccess = `
                                                           __
     _ _    _           _                                _|  |
 ___| | |  | |_ ___ ___| |_ ___    ___ ___ ___ ___ ___ _| |  |
| .'| | |  |  _| -_|_ -|  _|_ -|  | . | .'|_ -|_ -| -_| . |__|
|__,|_|_|  |_| |___|___|_| |___|  |  _|__,|___|___|___|___|__|
                                  |_|
`

	messageError = `
                     __
                    |  |
 ___ ___ ___ ___ ___|  |
| -_|  _|  _| . |  _|__|
|___|_| |_| |___|_| |__|

`
)

var (
	// list of arches to support.
	archList = []string{"amd64", "arm", "arm64", "ppc64le", "s390x"}
	// status of images is cached here, so that the same image is not
	// tested by multiple tests.
	verifiedImageCache = make(map[string]error)
)

// bellow are some types as per the docker specs.

type archContents struct {
	Architecture string `json:"architecture"`
}

type imageLayer struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
}

type manifestImage struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	Config        imageLayer   `json:"config"`
	Layers        []imageLayer `json:"layers"`
}

type manifest struct {
	MediaType string `json:"mediaType"`
	Size      int    `json:"size"`
	Digest    string `json:"digest"`
	Platform  struct {
		Architecture string `json:"architecture"`
		OS           string `json:"os"`
	} `json:"platform"`
}

type manifestList struct {
	SchemaVersion int        `json:"schemaVersion"`
	MediaType     string     `json:"mediaType"`
	Manifests     []manifest `json:"manifests"`
}

// download progress tracking.

type writeCounter struct {
	total       int64
	totalLength int64
}

func (wc *writeCounter) PrintProgress() {
	if wc.totalLength == 0 {
		fmt.Printf("\r* progress...%d bytes", wc.total)
		return
	}
	fmt.Printf("\r* progress...%d %% ", int64((float64(wc.total)/float64(wc.totalLength))*100.0))
}

func (wc *writeCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.total += int64(n)
	if downloadLayers {
		wc.PrintProgress()
	}
	return n, nil
}

// VersionList is used for sorting a slice of Version structs.
type VersionList []*version.Version

func (s VersionList) print() {
	if len(s) == 0 {
		fmt.Println("[empty list]")
	}
	for i, v := range s {
		sep := ", "
		if i == len(s)-1 {
			sep = "\n\n"
		}
		fmt.Printf(v.String() + sep)
	}
}

func (s VersionList) sort() {
	sort.Sort(s)
}

func (s VersionList) Len() int {
	return len(s)
}

func (s VersionList) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s VersionList) Less(i, j int) bool {
	return s[j].LessThan(s[i])
}

// throw an error.
func exitWithError(err error) {
	fmt.Printf("\n* ERROR: %v\n\n", err)
	fmt.Printf(messageError)
	os.Exit(1)
}

// prints a long line of characters.
func printLineSeparator(r rune) {
	fmt.Println(strings.Repeat(string(r), 79))
}

// downloads the contents of a web page into a string.
// use default timeout of 10 seconds.
func getFromURL(url string) (string, int, error) {
	return getFromURLTimeoutSize(url, defaultHTTPTimeout, false)
}

func getFromURLTimeoutSize(url string, timeout int, sizeOnly bool) (string, int, error) {
	fmt.Printf("* getFromURL(): %s\n", url)

	t := time.Duration(time.Duration(timeout) * time.Second)
	client := http.Client{
		Timeout: t,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", -1, err
	}
	req.Header.Set("Accept", "*/*")

	resp, err := client.Do(req)
	if err != nil {
		return "", -1, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", -1, fmt.Errorf("responded with status: %d", resp.StatusCode)
	}

	len := resp.Header.Get("Content-Length")
	sz, err := strconv.Atoi(len)
	if err != nil {
		fmt.Printf("WARNING: could not covert 'Content-Length' %q to integer\n", len)
		sz = 0
	}
	// only the Content-Length is requested; do not download the whole file.
	if sizeOnly {
		return "", sz, nil
	}

	var src io.Reader
	var dst bytes.Buffer

	counter := &writeCounter{totalLength: int64(sz)}
	src = io.TeeReader(resp.Body, counter)

	_, err = io.Copy(&dst, src)
	if err != nil {
		exitWithError(err)
	}
	if downloadLayers {
		fmt.Println()
	}

	return dst.String(), sz, nil
}

func getKubeadmConstants(releaseBranch string) (string, error) {
	url := fmt.Sprintf("https://raw.githubusercontent.com/kubernetes/kubernetes/%s/cmd/kubeadm/app/constants/constants.go", releaseBranch)
	constants, _, err := getFromURL(url)
	if err != nil {
		return "", err
	}
	return constants, nil
}

// parse the kubeadm config and obtain image versions that are not bound to the k8s version.
func getImageVersions(ver *version.Version, images map[string]string) error {
	branch := fmt.Sprintf("release-%d.%d", ver.Major(), ver.Minor())
	constants, err := getKubeadmConstants(branch)
	if err != nil {
		// the branch might not exist yet (e.g. for alpha releases),
		// as a release-xx branch is only created after a beta release is cut.
		// fallback to "master" in such a case.
		fmt.Printf("* getImageVersions(): WARNING: branch %q seems to be missing; falling back to \"master\"\n", branch)
		constants, err = getKubeadmConstants("master")
		if err != nil {
			return err
		}
	}
	lines := strings.Split(constants, "\n")

	// create a map of all required images
	k8sVersionV := "v" + ver.String()
	images["kube-apiserver"] = k8sVersionV
	images["kube-controller-manager"] = k8sVersionV
	images["kube-scheduler"] = k8sVersionV
	images["kube-proxy"] = k8sVersionV
	images["etcd"] = ""
	images["pause"] = ""

	// images outside the scope of kubeadm, but still using the k8s version

	// the hyperkube image was removed for version v1.17
	if ver.Major() == 1 && ver.Minor() < 17 {
		images["hyperkube"] = k8sVersionV
	}
	// the cloud-controller-manager image was removed for version v1.16
	if ver.Major() == 1 && ver.Minor() < 16 {
		images["cloud-controller-manager"] = k8sVersionV
	}
	// test the conformance image, but only for newer versions as it was added in v1.13.0-alpha.2
	// also skip v1.21.0-beta.1 due to a bug that caused this image tag to not be released.
	conformanceMinVer := version.MustParseSemantic("v1.13.0-alpha.2")
	is21beta1, _ := ver.Compare("v1.21.0-beta.1")
	if ver.AtLeast(conformanceMinVer) && is21beta1 != 0 {
		images["conformance"] = k8sVersionV
	}

	// coredns changed image location after 1.21.0-alpha.1
	coreDNSNewVer := version.MustParseSemantic("v1.21.0-alpha.1")
	coreDNSPath := "coredns"
	if ver.AtLeast(coreDNSNewVer) {
		coreDNSPath = "coredns/coredns"
	}

	// parse the constants file and fetch versions.
	// note: Split(...)[1] is safe here given a line contains the key.
	for _, line := range lines {
		if strings.Contains(line, "CoreDNSVersion = ") {
			line = strings.TrimSpace(line)
			line = strings.Split(line, "CoreDNSVersion = ")[1]
			line = strings.Replace(line, `"`, "", -1)
			images[coreDNSPath] = line
		} else if strings.Contains(line, "DefaultEtcdVersion = ") {
			line = strings.TrimSpace(line)
			line = strings.Split(line, "DefaultEtcdVersion = ")[1]
			line = strings.Replace(line, `"`, "", -1)
			images["etcd"] = line
		} else if strings.Contains(line, "PauseVersion = ") {
			line = strings.TrimSpace(line)
			line = strings.Split(line, "PauseVersion = ")[1]
			line = strings.Replace(line, `"`, "", -1)
			images["pause"] = line
		}
	}
	// hardcode the tag for pause as older k8s branches lack a constant.
	if images["pause"] == "" {
		images["pause"] = "3.1"
	}
	// verify.
	fmt.Printf("* getImageVersions(): [%s] %#v\n", ver.String(), images)
	if images[coreDNSPath] == "" || images["etcd"] == "" {
		return fmt.Errorf("at least one image version could not be set: %#v", images)
	}
	return nil
}

// verify an image manifest and it's layers.
func verifyArchImage(arch, imageName, archImage string) error {
	// parse the arch image.
	image := manifestImage{}
	if err := json.Unmarshal([]byte(archImage), &image); err != nil {
		return fmt.Errorf("could not unmarshal arch image: %v", err)
	}

	if image.MediaType != typeManifest {
		return fmt.Errorf("unknown media type: %s, manifest: %#v", image.MediaType, image)
	}
	if image.SchemaVersion != 2 {
		return errors.New("manifest is not schemaVersion 2")
	}
	if len(image.Layers) == 0 {
		return fmt.Errorf("no layers for image %#v", image)
	}

	// download the config blob.
	if image.Config.Digest == "" {
		return fmt.Errorf("empty digest for image config: %#v", image.Config)
	}
	url := fmt.Sprintf("%s/%s/blobs/%s", registryBucket, imageName, image.Config.Digest)
	configBlob, _, err := getFromURL(url)
	if err != nil {
		return fmt.Errorf("cannot download image blob for digest %q: %v", image.Config.Digest, err)
	}

	// verify the blob size.
	sz := len(configBlob)
	if image.Config.Size != sz {
		return fmt.Errorf("config size and image blob size differ for digest %q; wanted: %d, got: %d", image.Config.Digest, image.Config.Size, sz)
	}

	// verify the architecture in the config blob
	contents := archContents{}
	if err := json.Unmarshal([]byte(configBlob), &contents); err != nil {
		return fmt.Errorf("could not unmarshal config blob contents: %v", err)
	}
	if contents.Architecture != arch {
		// TODO(neolit123): consider making this an error at some point
		// https://github.com/kubernetes/kubernetes/issues/98908
		fmt.Printf("WARNING: in config digest %s: found architecture %q, expected %q\n", image.Config.Digest, contents.Architecture, arch)
	}

	// verify layers.
	for i, layer := range image.Layers {
		// only support the type defined in `typeLayer`?
		if layer.MediaType != typeLayer {
			return fmt.Errorf("not a layer: %s", layer.MediaType)
		}
		if layer.Digest == "" {
			return fmt.Errorf("empty digest for layer: %#v", layer)
		}

		// download layer blob and verify size.
		szMB := layer.Size / 1000000
		szMBStr := "<1 MB"
		if szMB > 0 {
			szMBStr = fmt.Sprintf("~%d MB", szMB)
		}
		if downloadLayers {
			fmt.Printf("* verifyArchImage(): downloading layer %d; size: %d bytes (%s)...\n", i+1, layer.Size, szMBStr)
		} else {
			fmt.Printf("* verifyArchImage(): downloading the header of layer %d; size: %d bytes\n", i+1, layer.Size)
		}

		url = fmt.Sprintf("%s/%s/blobs/%s", registryBucket, imageName, layer.Digest)
		layerBlob, sz, err := getFromURLTimeoutSize(url, defaultHTTPTimeout, !downloadLayers)
		if err != nil {
			return fmt.Errorf("cannot download layer blob for digest %q: %v", layer.Digest, err)
		}
		if downloadLayers {
			sz = len(layerBlob)
		}
		if layer.Size != sz {
			return fmt.Errorf("layer size differs; wanted: %d, got: %d", layer.Size, sz)
		}
	}

	return nil
}

// verify a manifest list and match the require architectures.
func verifyManifestList(manifest, imageName, tag string) error {
	ml := manifestList{}
	if err := json.Unmarshal([]byte(manifest), &ml); err != nil {
		return err
	}

	if ml.SchemaVersion != 2 {
		return errors.New("manifest is not schemaVersion 2")
	}
	if ml.MediaType != typeManifestList {
		return fmt.Errorf("not a manifest list: %s", ml.MediaType)
	}
	aList := make([]string, len(archList))
	// copy into a temp slice.
	copy(aList, archList)
	// traverse the manifests in the list.
	for _, m := range ml.Manifests {
		// skip unknown arches
		known := false
		for _, arch := range archList {
			if arch != m.Platform.Architecture {
				continue
			}
			known = true
			break
		}
		if !known {
			fmt.Printf("skipping unknown arch: %s\n", m.Platform.Architecture)
			continue
		}

		// verify media type and digest.
		if m.MediaType != typeManifest {
			return fmt.Errorf("unknown media type: %s, manifest: %#v", m.MediaType, m)
		}
		if m.Digest == "" {
			return fmt.Errorf("empty digest for manifest: %#v", m)
		}

		printLineSeparator('-')
		fmt.Printf("* verifyManifestList(): verifying image: %s-%s:%s\n", imageName, m.Platform.Architecture, tag)

		// match all required arches.
		for i, arch := range aList {
			if m.Platform.Architecture != arch {
				continue
			}
			aList[i] = aList[0] // remove i-th element.
			aList = aList[1:]
		}

		// download the arch minifest and verify its size.
		url := fmt.Sprintf("%s/%s/manifests/%s", registryBucket, imageName, m.Digest)
		archImageSrc, _, err := getFromURL(url)
		if err != nil {
			return fmt.Errorf("cannot download manifest for arch %q: %v", m.Platform.Architecture, err)
		}
		sz := len(archImageSrc)
		if m.Size != sz {
			return fmt.Errorf("manifest size differs for arch %q; wanted: %d, got: %d", m.Platform.Architecture, m.Size, sz)
		}

		// verify the arch image.
		err = verifyArchImage(m.Platform.Architecture, imageName, archImageSrc)
		if err != nil {
			return err
		}
	}

	if len(aList) > 0 {
		return fmt.Errorf("did not find a match for these architectures: %s", strings.Join(aList, ", "))
	}

	return nil
}

// verify all images for a given k8s version.
func verifyKubernetesVersion(ver *version.Version) ([]string, error) {
	missingImages := []string{}

	images := make(map[string]string)
	if err := getImageVersions(ver, images); err != nil {
		return missingImages, err
	}

	keys := make([]string, 0, len(images))
	for k := range images {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// download and process a manifest for each image:tag.
	for _, k := range keys {
		printLineSeparator('=')
		imageTag := fmt.Sprintf("%s:%s", k, images[k])
		fmt.Printf("* verifyManifestList(): %s\n", imageTag)

		url := fmt.Sprintf("%s/%s/manifests/%s", registryBucket, k, images[k])
		manifest, _, err := getFromURL(url)

		if err != nil {
			fmt.Printf("* ERROR: %v\n", err)
			missingImages = append(missingImages, imageTag)
			continue
		}

		// attempt to fetch result from cache.
		if err, ok := verifiedImageCache[imageTag]; ok {
			if err != nil {
				fmt.Printf("\n* ERROR(cached result): %s; error: %v\n", imageTag, err)
				missingImages = append(missingImages, imageTag)
			} else {
				fmt.Printf("\n* PASSED(cached result): %s\n", imageTag)
			}
			continue
		}

		// uncached; run tests
		if err = verifyManifestList(manifest, k, images[k]); err != nil {
			fmt.Printf("\n* ERROR: %s; error: %v\n", imageTag, err)
			missingImages = append(missingImages, imageTag)
			verifiedImageCache[imageTag] = err
		} else {
			fmt.Printf("\n* PASSED: %s\n", imageTag)
			verifiedImageCache[imageTag] = nil
		}
	}

	return missingImages, nil
}

// gets tags from github and parses a list of tags.
func getReleaseVersions() (VersionList, error) {
	const tagName = `"tag_name": "`
	tags := []string{}
	versions := VersionList{}

	releases, _, err := getFromURL("https://api.github.com/repos/kubernetes/kubernetes/releases")
	if err != nil {
		return versions, err
	}

	// format json so that line splits are doable.
	var formatted bytes.Buffer
	err = json.Indent(&formatted, []byte(releases), "", "\t")
	if err != nil {
		return versions, err
	}
	releases = formatted.String()

	// get list of tags.
	lines := strings.Split(releases, "\n")
	for _, line := range lines {
		if !strings.Contains(line, tagName) {
			continue
		}
		tag := strings.TrimSpace(line)
		tag = strings.Replace(tag, tagName, "", -1)
		tag = strings.Replace(tag, `",`, "", -1)
		tags = append(tags, tag)
	}

	// convert tags to versions.
	for _, tag := range tags {
		ver, err := version.ParseSemantic(tag)
		if err != nil {
			return versions, err
		}
		versions = append(versions, ver)
	}

	versions.sort()
	return versions, nil
}

// this function filters the list of releases so that only has supported releases.
func filterVersions(versions VersionList) (VersionList, error) {
	if len(versions) == 0 {
		fmt.Println("* WARNING: no versions to filter; the list is empty")
		return versions, nil
	}

	// the first version when manifest lists where fully supported
	min := version.MustParseSemantic(firstKnownVersion)

	// based on the max version (top of the list) find the version for the support skew
	// e.g. 1.X -> 1.[X - 2]. discard pre-release and meta data.
	semVersion := fmt.Sprintf("%d.%d.%d", versions[0].Major(), versions[0].Minor(), versions[0].Patch())
	minEstimated := version.MustParseSemantic(semVersion)

	// if the Minor is bigger enough apply the skew.
	if minEstimated.Minor() > 1 {
		minor := minEstimated.Minor() - 2
		semVersion := fmt.Sprintf("%d.%d.%d", minEstimated.Major(), minor, minEstimated.Patch())
		minEstimated = version.MustParseGeneric(semVersion)
		// else get the last version from the previous Major release and apply the skew.
	} else {
		url := fmt.Sprintf("https://storage.googleapis.com/kubernetes-release/release/stable-%d.txt", minEstimated.Major()-1)
		verFromURL, _, err := getFromURL(url)
		if err != nil {
			return versions, err
		}
		verFromURL = strings.TrimSpace(verFromURL)
		minEstimated := version.MustParseGeneric(verFromURL)
		minor := minEstimated.Minor() - (2 - minEstimated.Minor())
		semVersion := fmt.Sprintf("%d.%d.%d", minEstimated.Major(), minor, minEstimated.Patch())
		minEstimated = version.MustParseGeneric(semVersion)
	}
	// if minEstimated is bigger use that.
	if min.LessThan(minEstimated) {
		min = version.MustParseGeneric(minEstimated.String())
	}

	filteredVersions := VersionList{}
	for _, v := range versions {
		if !v.LessThan(min) { // only use versions equal or bigger than min.
			filteredVersions = append(filteredVersions, v)
		}
	}

	filteredVersions.sort()
	return filteredVersions, nil
}

func main() {
	printLineSeparator('#')
	fmt.Println(messageStart)
	fmt.Println("** kubeadm manifest list verification tests **")
	fmt.Printf("\nrequired architectures:\n%s\n\n", strings.Join(archList, ", "))

	// download tags from github.
	versions, err := getReleaseVersions()
	if err != nil {
		exitWithError(err)
	}

	// print downloaded versions.
	fmt.Println("* downloaded the following releases from GitHub:")
	versions.print()

	// filter the versions.
	filteredVersions, err := filterVersions(versions)
	if err != nil {
		exitWithError(err)
	}
	fmt.Println("* testing the following list of releases that should support manifest lists:")
	filteredVersions.print()

	// process versions.
	versionsWithErrors := []string{}
	for _, v := range filteredVersions {
		fmt.Println()
		printLineSeparator('#')
		fmt.Printf("verify k8s %s\n", v.String())
		printLineSeparator('#')
		fmt.Println()

		missingImages, err := verifyKubernetesVersion(v)
		if err != nil {
			fmt.Printf("\n* ERROR: could not process version %q: %s\n", v.String(), err)
			continue
		}
		if len(missingImages) > 0 {
			fmt.Printf("\n* ERROR: the following images have manifest lists errors for version %q: %s\n", v.String(), strings.Join(missingImages, ", "))
			versionsWithErrors = append(versionsWithErrors, v.String())
		} else {
			fmt.Printf("\n* PASSED: all image checks passed for: %s\n", v.String())
		}
	}

	// print outcome.
	if len(versionsWithErrors) > 0 {
		exitWithError(fmt.Errorf("the following k8s versions have manifest lists errors: %s", strings.Join(versionsWithErrors, ", ")))
	} else {
		fmt.Println(messageSuccess)
	}
}
