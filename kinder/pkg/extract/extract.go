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
Package extract implements support for extracting required K8s binaries and required K8s images
from GCS buckets containing release or ci builds artifacts.

Additionally, it is also possible to manage local repositories of the aforementioned artifacts
or repository hosted on http/https web servers.
*/
package extract

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	K8sVersion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/apimachinery/pkg/util/wait"
	kindfs "sigs.k8s.io/kind/pkg/fs"
)

const (
	ciBuildRepository       = "https://storage.googleapis.com/k8s-release-dev/ci"
	releaseBuildURepository = "https://dl.k8s.io/release"

	kubeadmBinary = "kubeadm"
	kubeletBinary = "kubelet"
	kubectlBinary = "kubectl"
)

var (
	allKubernetesBinaries = []string{kubeletBinary, kubectlBinary, kubeadmBinary}

	// AllKubernetesImages defines of image tarballs included in a K8s release
	AllKubernetesImages = []string{"kube-apiserver.tar", "kube-controller-manager.tar", "kube-scheduler.tar", "kube-proxy.tar"}

	// AllImagesPattern defines a pattern for searching all the images in a folder
	AllImagesPattern = []string{"*.tar"}
)

// SourceType defines src types
type SourceType int

const (
	// ReleaseLabelOrVersionSource describe a src that is hosted in releaseBuildURepository
	ReleaseLabelOrVersionSource SourceType = iota + 1

	// CILabelOrVersionSource describe a src that is hosted in ciBuildRepository
	CILabelOrVersionSource

	// RemoteRepositorySource describe a src that is hosted in a remote http/https repository
	RemoteRepositorySource

	// LocalRepositorySource describe a src that is hosted in local repository
	LocalRepositorySource
)

// GetSourceType returns the src type descriptor
func GetSourceType(src string) SourceType {
	if strings.HasPrefix(src, "file://") {
		return LocalRepositorySource
	} else if strings.HasPrefix(src, "release/") {
		return ReleaseLabelOrVersionSource
	} else if strings.HasPrefix(src, "ci/") {
		return CILabelOrVersionSource
	} else if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return RemoteRepositorySource
	} else if v, err := K8sVersion.ParseSemantic(src); err == nil {
		if v.BuildMetadata() != "" {
			return CILabelOrVersionSource
		}
		return ReleaseLabelOrVersionSource
	}
	return LocalRepositorySource
}

// Option is an Extractor configuration option supplied to NewExtractor
type Option func(*Extractor)

// OnlyKubeadm option instructs the Extractor for retrieving kubeadm only
func OnlyKubeadm(onlyKubeadm bool) Option {
	return func(b *Extractor) {
		if onlyKubeadm {
			b.files = []string{kubeadmBinary}
			// disable addVersionFileToDst when we are reading only a subset of files
			b.addVersionFileToDst = false
		}
	}
}

// OnlyKubelet option instructs the Extractor for retrieving kubelet only
func OnlyKubelet(onlyKubelet bool) Option {
	return func(b *Extractor) {
		if onlyKubelet {
			b.files = []string{kubeletBinary}
			// disable addVersionFileToDst when we are reading only a subset of files
			b.addVersionFileToDst = false
		}
	}
}

// OnlyKubernetesBinaries option instructs the Extractor for retrieving Kubernetes binaries only
func OnlyKubernetesBinaries(onlyBinaries bool) Option {
	return func(b *Extractor) {
		if onlyBinaries {
			b.files = allKubernetesBinaries
			// disable addVersionFileToDst when we are reading only a subset of files
			b.addVersionFileToDst = false
		}
	}
}

// OnlyKubernetesImages option instructs the Extractor for retrieving Kubernetes images tarballs only
func OnlyKubernetesImages(onlyImages bool) Option {
	return func(b *Extractor) {
		if onlyImages {
			b.files = AllKubernetesImages
			// disable addVersionFileToDst when we are reading only a subset of files
			b.addVersionFileToDst = false
		}
	}
}

// WithNamePrefix option instructs the Extractor to adds a prefix to the name of each file before saving to destination
func WithNamePrefix(namePrefix string) Option {
	return func(b *Extractor) {
		b.dstMutator.namePrefix = namePrefix
	}
}

// WithNameOverride option instructs the Extractor to rename the bit
func WithNameOverride(newName string) Option {
	return func(b *Extractor) {
		b.dstMutator.nameOverride = newName
	}
}

// WithVersionFile option instructs the Extractor whether to add version file
func WithVersionFile(enable bool) Option {
	return func(b *Extractor) {
		b.addVersionFileToDst = enable
	}
}

// WithVersionFolder option instructs the Extractor to save all files in a folder named like the kubernetes version
func WithVersionFolder(versionFolder bool) Option {
	return func(b *Extractor) {
		b.dstMutator.prependVersionFolder = versionFolder
	}
}

// Extractor defines attributes for a Kubernetes artifact extractor
type Extractor struct {
	// src is the source from where to extract file
	src string
	// files is the list of files to extract
	files []string
	// dst folder
	dst string
	// dst file name mutator
	dstMutator fileNameMutator
	// add version file to dst
	addVersionFileToDst bool
}

// NewExtractor returns a new extractor configured with the given options
func NewExtractor(src, dst string, options ...Option) (extractor *Extractor) {
	extractor = &Extractor{
		src:                 src,
		files:               append(allKubernetesBinaries, AllKubernetesImages...),
		dst:                 dst,
		dstMutator:          fileNameMutator{},
		addVersionFileToDst: true,
	}

	// apply user options
	for _, option := range options {
		option(extractor)
	}

	return extractor
}

// SetFiles allows to override the default list of files that the extractor is expected to retrieve.
func (e *Extractor) SetFiles(files []string) {
	e.files = files
}

// Extract Kubernetes artifacts from the given source
func (e *Extractor) Extract() (paths map[string]string, err error) {
	var f extractFunc

	switch GetSourceType(e.src) {
	case ReleaseLabelOrVersionSource:
		f = extractFromReleaseBuild
	case CILabelOrVersionSource:
		f = extractFromCIBuild
	case RemoteRepositorySource:
		f = extractFromHTTP
	case LocalRepositorySource:
		f = extractFromLocalDir
	default:
		return nil, errors.Errorf("source %s did not resolve to a valid source type", e.src)
	}

	return f(e.src, e.files, e.dst, e.dstMutator, e.addVersionFileToDst)
}

// extractFunc define a function that implements an extractor method
type extractFunc func(string, []string, string, fileNameMutator, bool) (map[string]string, error)

func extractFromCIBuild(src string, files []string, dst string, m fileNameMutator, addVersionFileToDst bool) (paths map[string]string, err error) {
	// cleanup the src from the prefix, if any
	src = strings.TrimPrefix(src, "ci/")

	// gets the Kubernetes version from the src
	version, err := K8sVersion.ParseSemantic(src)
	if err != nil {
		version, err = resolveLabel(ciBuildRepository, src)
		if err != nil {
			return nil, err
		}
	}

	// saves the version file (if requested)
	// nb. version file is created so the target folder can be eventually used as a source
	if err := saveVersionFile(addVersionFileToDst, dst, version, m); err != nil {
		return nil, errors.Wrapf(err, "error creating version file in %s", dst)
	}

	// pass the version to the file name mutator
	// nb. this will allow to save extracted files into a version folder
	m.SetPrependVersionFolder(version)

	// sets the url for downloading the requested ci version
	src = fmt.Sprintf("%s/v%s", ciBuildRepository, version)

	// read from the src via http, taking care of setting addVersionFileToDst (because it was already saved above)
	return extractFromHTTP(src, files, dst, m, false)
}

func extractFromReleaseBuild(src string, files []string, dst string, m fileNameMutator, addVersionFileToDst bool) (paths map[string]string, err error) {
	// cleanup the source src the prefix, if any
	src = strings.TrimPrefix(src, "release/")

	// gets the Kubernetes version from the src
	version, err := K8sVersion.ParseSemantic(src)
	if err != nil {
		version, err = resolveLabel(releaseBuildURepository, src)
		if err != nil {
			return nil, err
		}
	}

	// saves the version file (if requested)
	// nb. version file is created so the target folder can be eventually used as a source
	if err := saveVersionFile(addVersionFileToDst, dst, version, m); err != nil {
		return nil, errors.Wrapf(err, "error creating version file in %s", dst)
	}

	// pass the version to the file name mutator
	// nb. this will allow to save extracted files into a version folder
	m.SetPrependVersionFolder(version)

	// sets the url for downloading the requested release version
	src = fmt.Sprintf("%s/v%s", releaseBuildURepository, version)

	// read from the src via http, taking care of setting addVersionFileToDst (because it was already saved above)
	return extractFromHTTP(src, files, dst, m, false)
}

func extractFromHTTP(src string, files []string, dst string, m fileNameMutator, addVersionFileToDst bool) (paths map[string]string, err error) {
	dst, _ = filepath.Abs(dst)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return nil, errors.Errorf("destination path %s does not exists", dst)
	}

	// ensure folder required by the fileNameMutator exist
	// nb. this will allow to save extracted files into a version folder
	if err := m.EnsureFolder(dst); err != nil {
		return nil, err
	}

	// if required, add the version file to the list of files to be copied to dest
	// nb. version file is created so the target folder can be eventually used as a source
	if addVersionFileToDst {
		files = append(files, "version")
	}

	// in case the source is a Kubernetes build, add bin/OS/ARCH to the src uri
	if strings.HasPrefix(src, releaseBuildURepository) || strings.HasPrefix(src, ciBuildRepository) {
		src = fmt.Sprintf("%s/bin/linux/%s", src, runtime.GOARCH)
	}

	// Download the files.
	paths = map[string]string{}
	for _, f := range files {
		srcFilePath := fmt.Sprintf("%s/%s", src, f)
		log.Infof("Downloading %s\n", srcFilePath)
		dstFilePath := path.Join(dst, m.Mutate(f))
		if err := copyFromURI(srcFilePath, dstFilePath); err != nil {
			return nil, errors.Wrapf(err, "failed to copy %s to %s", srcFilePath, dstFilePath)
		}
		if f == kubeadmBinary || f == kubeletBinary || f == kubectlBinary {
			os.Chmod(dstFilePath, 0755)
		}
		paths[f] = dstFilePath
	}
	log.Infof("Downloaded files saved into %s", dst)

	return paths, nil
}

func extractFromLocalDir(src string, files []string, dst string, m fileNameMutator, addVersionFileToDst bool) (paths map[string]string, err error) {
	// checks if source folder exists
	src, _ = filepath.Abs(src)
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil, errors.Errorf("source path %s does not exists", src)
	}

	// read version file (only if required by the fileNameMutator)
	if err := m.ReadVersionFile(src); err != nil {
		return nil, err
	}

	// checks if target folder exists
	dst, _ = filepath.Abs(dst)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return nil, errors.Errorf("destination path %s does not exists", dst)
	}

	// ensure folder required by the fileNameMutator exist (if any)
	// nb. this will allow to save extracted files into a version folder
	if err := m.EnsureFolder(dst); err != nil {
		return nil, err
	}

	// if the local repository is a single file
	if info, err := os.Stat(src); err == nil && !info.IsDir() {
		// sets the extractor for getting this file only overriding the default file list
		files = []string{filepath.Base(src)}

		// points the extractor to the upper folder (the extractor expects a folder)
		parent := filepath.Dir(src)
		log.Debugf("%s is a file, moving up one level to %s", src, parent)
		src = parent
	} else {
		// Expanding wildcards defined in the list of files (if any)
		// NB. this is required because for imageBits we want to allow to extract
		// all the images in a folder, not only the Kubernetes one
		files, err = expandWildcards(src, files)
		if err != nil {
			return nil, err
		}
	}

	// if required, add the version file to the list of files to be copied to dest
	// nb. version file is created so the target folder can be eventually used as a source
	if addVersionFileToDst {
		files = append(files, "version")
	}

	// copy files from source to target
	paths = map[string]string{}
	for _, f := range files {
		srcFilePath := path.Join(src, f)
		if _, err := os.Stat(srcFilePath); err != nil {
			return nil, errors.Wrapf(err, "cannot access %s at %s", f, srcFilePath)
		}
		log.Infof("Copying %s", srcFilePath)

		dstFilePath := path.Join(dst, m.Mutate(f))
		// NOTE: we use copy not copyfile because copy ensures the dest dir
		if err := kindfs.Copy(srcFilePath, dstFilePath); err != nil {
			return nil, errors.Wrap(err, "failed to copy alter bits")
		}
		if f == kubeadmBinary || f == kubeletBinary || f == kubectlBinary {
			os.Chmod(dstFilePath, 0755)
		}
		paths[f] = dstFilePath
	}

	log.Infof("Copied files saved into %s", dst)
	return paths, nil
}

func expandWildcards(src string, files []string) (expandedFiles []string, err error) {
	for _, f := range files {
		switch {
		case strings.Contains(f, "*"):
			// if file name is a wildcard, search matching files and add them to
			// the list of files to extract
			log.Debugf("searching source with wildcard %s\n", f)
			pattern := filepath.Join(src, f)
			matches, err := filepath.Glob(pattern)
			if err != nil {
				return nil, errors.Errorf("invalid pattern %s", pattern)
			}

			log.Debugf("%d matches found\n", len(matches))
			for _, f := range matches {
				expandedFiles = append(expandedFiles, filepath.Base(f))
			}
		default:
			// otherwise the file name is an actual file to extract; preserve it
			expandedFiles = append(expandedFiles, f)
		}
	}

	return expandedFiles, nil
}

func resolveLabel(repository, label string) (version *K8sVersion.Version, err error) {
	// labels are .txt file containing a release version

	// Gets the uri of the label file
	uri := fmt.Sprintf("%s/%s", repository, label)
	if !strings.HasSuffix(uri, ".txt") {
		uri = uri + ".txt"
	}
	log.Debugf("Resolving label %s\n", uri)

	// Do an HTTP GET and read the version from the txt file.
	_, r, err := httpGet(uri)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid version URI: %s", uri)
	}
	defer r.Close()

	version, err = readVersion(r)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading version from %s", uri)
	}

	log.Debugf("Label %s resolves to v%s\n", uri, version)
	return version, nil
}

func readVersion(r io.Reader) (version *K8sVersion.Version, err error) {
	buf, err := io.ReadAll(r)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading version")
	}

	labelValue := url.PathEscape(string(bytes.TrimSpace(buf)))
	version, err = K8sVersion.ParseSemantic(labelValue)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid version")
	}

	return version, nil
}

func saveVersionFile(addVersionFileToDst bool, dst string, version *K8sVersion.Version, m fileNameMutator) error {
	if !addVersionFileToDst {
		return nil
	}

	versionFile := filepath.Join(dst, m.Mutate("version"))
	f, err := os.Create(versionFile)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(fmt.Sprintf("v%s", version))
	if err != nil {
		return err
	}

	log.Info("version file created")

	return nil
}

// Exponential backoff for httpGet (values exclude jitter):
// 0, 2, 5, 8 ... 322 s
var httpGetBackoff = wait.Backoff{
	Steps:    20,
	Duration: 2000 * time.Millisecond,
	Factor:   1.2,
	Jitter:   0.1,
}

func httpGet(uri string) (int64, io.ReadCloser, error) {
	var lastError error
	var resp *http.Response

	// Create a custom http.Client with redirect behavior
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// Allow redirects
			return nil
		},
	}

	err := wait.ExponentialBackoff(httpGetBackoff, func() (bool, error) {
		var err error
		resp, err = client.Get(uri)
		if err != nil {
			log.Warnf("HTTP GET %s failed. Retry in few seconds", uri)
			lastError = errors.Wrapf(err, "HTTP GET %s failed", uri)
			return false, nil
		}
		if resp.StatusCode != http.StatusOK {
			log.Warnf("HTTP GET %s failed: %s. Retry in few seconds", uri, resp.Status)
			lastError = errors.Errorf("HTTP GET %s failed: %s", uri, resp.Status)
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		return 0, nil, lastError
	}

	return resp.ContentLength, resp.Body, nil
}

func copyFromURI(src, dst string) error {
	size, r, err := httpGet(src)
	if err != nil {
		return errors.Wrapf(err, "error getting reader for %s", src)
	}
	defer r.Close()

	// If the file already exists and has the same size as the remote
	// content then do not redownload it.
	if f, err := os.Stat(dst); err == nil {
		if size == f.Size() {
			return nil
		}
	}

	w, err := os.Create(dst)
	if err != nil {
		return errors.Wrapf(err, "error creating %s", dst)
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return errors.Wrapf(err, "error copying %s to %s", src, dst)
	}

	return nil
}

type fileNameMutator struct {
	nameOverride         string
	namePrefix           string
	prependVersionFolder bool
	prependFolder        string
}

func (m *fileNameMutator) Mutate(name string) string {
	if name == "version" {
		return name
	}

	if m.nameOverride != "" {
		return m.nameOverride
	}

	if m.namePrefix != "" {
		name = fmt.Sprintf("%s-%s", m.namePrefix, name)
	}
	if m.prependFolder != "" {
		name = filepath.Join(m.prependFolder, name)
	}
	return name
}

func (m *fileNameMutator) EnsureFolder(dst string) error {
	if m.prependFolder != "" {
		prependFolder := filepath.Join(dst, m.prependFolder)
		if err := os.Mkdir(prependFolder, 0777); err != nil {
			return errors.Wrapf(err, "failed to make %s dir", prependFolder)
		}
	}
	return nil
}

func (m *fileNameMutator) ReadVersionFile(src string) error {
	if m.prependVersionFolder {
		versionFile := filepath.Join(src, "version")
		if _, err := os.Stat(versionFile); os.IsNotExist(err) {
			return errors.Errorf("%s does not exists. please provide a version file", versionFile)
		}
		f, err := os.Open(versionFile)
		if err != nil {
			return errors.Wrapf(err, "error reading version from %s", versionFile)
		}
		version, err := readVersion(f)
		if err != nil {
			return errors.Wrapf(err, "error reading version from %s", versionFile)
		}

		m.SetPrependVersionFolder(version)
	}
	return nil
}

func (m *fileNameMutator) SetPrependVersionFolder(version *K8sVersion.Version) {
	if m.prependVersionFolder {
		m.prependFolder = fmt.Sprintf("v%s", version)
	}
}

// ResolveLabel provide a utility func for resolving a label
func ResolveLabel(src string) (version string, err error) {
	var repository string
	switch GetSourceType(src) {
	case ReleaseLabelOrVersionSource:
		src = strings.TrimPrefix(src, "release/")

		repository = releaseBuildURepository
	case CILabelOrVersionSource:
		src = strings.TrimPrefix(src, "ci/")

		repository = ciBuildRepository
	default:
		return "", errors.Errorf("source %s did not resolve to a valid label", src)
	}

	v, err := resolveLabel(repository, src)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("v%s", v.String()), nil
}
