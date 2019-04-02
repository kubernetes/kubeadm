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

// Package extract implements the `extract` library; please note that this
// packages should be replace by the same capability offered by kind/by a
// shared library as soon as available
package extract

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"

	versionutil "k8s.io/apimachinery/pkg/util/version"
)

const (
	ciBuildRepository       = "https://storage.googleapis.com/kubernetes-release-dev/ci"
	releaseBuildURepository = "https://storage.googleapis.com/kubernetes-release/release"
)

var (
	kubeadmBinary = "kubeadm"
	kubeletBinary = "kubelet"
	allBinaries   = append([]string{"kubelet", "kubectl"}, kubeadmBinary)
	allImages     = []string{"kube-apiserver.tar", "kube-controller-manager.tar", "kube-scheduler.tar", "kube-proxy.tar"}
)

type SourceType int

const (
	// ReleaseLabelOrVersionSource describe a src that should read from releaseBuildURepository
	ReleaseLabelOrVersionSource SourceType = iota + 1

	// CILabelOrVersionSource describe a src that should read from ciBuildRepository
	CILabelOrVersionSource

	// RemoteRepositorySource describe a src that should read from a remote http/https repository
	RemoteRepositorySource

	// LocalRepositorySource describe a src that should read from repository hosted in a local folder
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
	} else if v, err := versionutil.ParseSemantic(src); err == nil {
		if v.BuildMetadata() != "" {
			return CILabelOrVersionSource
		}
		return ReleaseLabelOrVersionSource
	}
	return LocalRepositorySource
}

// Option is an Extractor configuration option supplied to NewExtractor
type Option func(*Extractor)

// OnlyKubeadm option instructs the Extractor for retriving kubeadm only
func OnlyKubeadm(onlyKubeadm bool) Option {
	return func(b *Extractor) {
		if onlyKubeadm {
			b.files = []string{kubeadmBinary}
		}
	}
}

// OnlyKubelet option instructs the Extractor for retriving kubelet only
func OnlyKubelet(onlyKubelet bool) Option {
	return func(b *Extractor) {
		if onlyKubelet {
			b.files = []string{kubeletBinary}
		}
	}
}

// OnlyBinaries option instructs the Extractor for retriving kubeadm, kubectl and kubelet binaries only
func OnlyBinaries(onlyBinaries bool) Option {
	return func(b *Extractor) {
		if onlyBinaries {
			b.files = allBinaries
		}
	}
}

// OnlyImages option instructs the Extractor for retriving images tarballs only
func OnlyImages(onlyImages bool) Option {
	return func(b *Extractor) {
		if onlyImages {
			b.files = allImages
		}
	}
}

// Extractor defines attributes for a Kubernetes artifact extractor
type Extractor struct {
	// src is the source from where to extract file
	src string
	// files is the list of files to extract
	files []string
	dst   string
}

// NewExtractor returns a new extractor configured with the given options
func NewExtractor(src, dst string, options ...Option) (extractor *Extractor) {
	extractor = &Extractor{
		src:   src,
		dst:   dst,
		files: append(allBinaries, allImages...),
	}

	// apply user options
	for _, option := range options {
		option(extractor)
	}

	return extractor
}

// extractFunc define a function that implements an extractor method
type extractFunc func(string, []string, string) (map[string]string, error)

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
		errors.Errorf("source %s did not resolve to a valid source type", e.src)
	}

	return f(e.src, e.files, e.dst)
}

func extractFromCIBuild(src string, files []string, dst string) (paths map[string]string, err error) {
	// cleanup the src from the prefix, if any
	src = strings.TrimPrefix(src, "ci/")

	// gets the Kubernetes version from the src
	version, err := versionutil.ParseSemantic(src)
	if err != nil {
		version, err = resolveLabel(ciBuildRepository, src)
		if err != nil {
			return nil, err
		}
	}

	// sets the url for downloading the requested ci version and triggers the extraction
	src = fmt.Sprintf("%s/v%s", ciBuildRepository, version)
	return extractFromHTTP(src, files, dst)
}

func extractFromReleaseBuild(src string, files []string, dst string) (paths map[string]string, err error) {
	// cleanup the source src the prefix, if any
	src = strings.TrimPrefix(src, "release/")

	// gets the Kubernetes version from the src
	version, err := versionutil.ParseSemantic(src)
	if err != nil {
		version, err = resolveLabel(releaseBuildURepository, src)
		if err != nil {
			return nil, err
		}
	}

	// sets the url for downloading the requested release version and triggers the extraction
	src = fmt.Sprintf("%s/v%s", releaseBuildURepository, version)

	return extractFromHTTP(src, files, dst)
}

func extractFromHTTP(src string, files []string, dst string) (paths map[string]string, err error) {
	dst, _ = filepath.Abs(dst)
	if _, err := os.Stat(dst); os.IsNotExist(err) {
		return nil, errors.Errorf("destination path %s does not exists", dst)
	}

	// Build the URIs to the image archives and binaries.
	srcBinDirPath := fmt.Sprintf("%s/bin/linux/amd64", src)

	// Download the files.
	for _, f := range files {
		srcFilePath := fmt.Sprintf("%s/%s", srcBinDirPath, f)
		fmt.Printf("Downloading %s\n", srcFilePath)
		dstFilePath := path.Join(dst, f)
		if err := copyFromURI(srcFilePath, dstFilePath); err != nil {
			return nil, errors.Wrapf(err, "failed to copy %s to %s", srcFilePath, dstFilePath)
		}
	}

	return extractFromLocalDir(dst, files, dst)
}

func extractFromLocalDir(src string, files []string, dst string) (paths map[string]string, err error) {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return nil, errors.Errorf("source path %s does not exists", src)
	}

	paths = map[string]string{}
	for _, f := range files {
		srcFilePath := path.Join(src, f)
		if _, err := os.Stat(srcFilePath); err != nil {
			return nil, errors.Wrapf(err, "cannot access %s at %s", f, srcFilePath)
		}
		paths[f] = srcFilePath
	}

	return paths, nil
}

func resolveLabel(repository, label string) (version *versionutil.Version, err error) {
	// labels are .txt file containing a release version

	// Gets the uri of the label file
	uri := fmt.Sprintf("%s/%s", repository, label)
	if !strings.HasSuffix(uri, ".txt") {
		uri = uri + ".txt"
	}

	// Do an HTTP GET and read the version from the txt file.
	_, r, err := httpGet(uri)
	if err != nil {
		return nil, errors.Wrapf(err, "invalid version URI: %s", uri)
	}
	defer r.Close()
	buf, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading version from %s", uri)
	}

	labelValue := url.PathEscape(string(bytes.TrimSpace(buf)))
	version, err = versionutil.ParseSemantic(labelValue)
	if err != nil {
		return nil, errors.Wrapf(err, "label %s returned invalid version: %s", label, labelValue)
	}

	return version, nil
}

func httpGet(uri string) (int64, io.ReadCloser, error) {
	resp, err := http.Get(uri)
	if err != nil {
		return 0, nil, errors.Wrapf(err, "HTTP GET %s failed", uri)
	}
	if resp.StatusCode != http.StatusOK {
		return 0, nil, errors.Errorf("HTTP GET %s failed: %s", uri, resp.Status)
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
