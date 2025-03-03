/*
Copyright 2020 The Kubernetes Authors.

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

package kubeadm

import (
	"bufio"
	"bytes"
	"io"
	"strings"

	"github.com/pkg/errors"
	jsonpatch "gopkg.in/evanphx/json-patch.v4"
	"sigs.k8s.io/yaml"
)

/*
patch.go provides utilities for applying patches to a YAML file.

The current implementation is a fork from "https://sigs.k8s.io/kind/pkg/cluster/internal/patch",
which can't be used because it is internal. The code in this package is now evolving independently from the original.
*/

// PatchJSON6902 represents an inline kustomize json 6902 patch
// https://tools.ietf.org/html/rfc6902
type PatchJSON6902 struct {
	// these fields specify the patch target resource
	Group   string `json:"group"`
	Version string `json:"version"`
	Kind    string `json:"kind"`
	// Patch should contain the contents of the json patch as a string
	Patch string `json:"patch"`
}

// Build takes a Kubernetes object YAML document stream to patch,
// merge patches, and JSON 6902 patches.
//
// It returns a patched YAML document stream.
//
// Matching is performed on Kubernetes style v1 TypeMeta fields
// (kind and apiVersion), between the YAML documents and the patches.
//
// Patches match if their kind and apiVersion match a document, with the exception
// that if the patch does not set apiVersion it will be ignored.
func Build(toPatch string, patches []string, patches6902 []PatchJSON6902) (string, error) {
	// pre-process, including splitting up documents etc.
	resources, err := parseResources(toPatch)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse yaml to patch")
	}
	mergePatches, err := parseMergePatches(patches)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse patches")
	}
	json6902patches, err := convertJSON6902Patches(patches6902)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse JSON 6902 patches")
	}
	// apply patches and build result
	builder := &strings.Builder{}
	for i, r := range resources {
		// apply merge patches
		for _, p := range mergePatches {
			if _, err := r.applyMergePatch(p); err != nil {
				return "", errors.Wrap(err, "failed to apply patch")
			}
		}
		// apply RFC 6902 JSON patches
		for _, p := range json6902patches {
			if _, err := r.apply6902Patch(p); err != nil {
				return "", errors.Wrap(err, "failed to apply JSON 6902 patch")
			}
		}
		// write out result
		if err := r.encodeTo(builder); err != nil {
			return "", errors.Wrap(err, "failed to write patched resource")
		}
		// write document separator
		if i+1 < len(resources) {
			if _, err := builder.WriteString("---\n"); err != nil {
				return "", errors.Wrap(err, "failed to write document separator")
			}
		}
	}
	// verify that all patches were used
	return builder.String(), nil
}

type resource struct {
	raw       string    // the original raw data
	json      []byte    // the processed data (in JSON form), may be mutated
	matchInfo matchInfo // for matching patches
}

func (r *resource) apply6902Patch(patch json6902Patch) (matches bool, err error) {
	if !r.matches(patch.matchInfo) {
		return false, nil
	}
	patched, err := patch.patch.Apply(r.json)
	if err != nil {
		return true, errors.WithStack(err)
	}
	r.json = patched
	return true, nil
}

func (r *resource) applyMergePatch(patch mergePatch) (matches bool, err error) {
	if !r.matches(patch.matchInfo) {
		return false, nil
	}
	patched, err := jsonpatch.MergePatch(r.json, patch.json)
	if err != nil {
		return true, errors.WithStack(err)
	}
	r.json = patched
	return true, nil
}

func (r resource) matches(o matchInfo) bool {
	m := &r.matchInfo
	// we require kind to match, but if the patch does not specify
	// APIVersion we ignore it (eg to allow trivial patches across kubeadm versions)
	return m.Kind == o.Kind && (o.APIVersion == "" || m.APIVersion == o.APIVersion)
}

func (r *resource) encodeTo(w io.Writer) error {
	encoded, err := yaml.JSONToYAML(r.json)
	if err != nil {
		return errors.WithStack(err)
	}
	if _, err := w.Write(encoded); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func parseResources(yamlDocumentStream string) ([]resource, error) {
	resources := []resource{}
	documents, err := splitYAMLDocuments(yamlDocumentStream)
	if err != nil {
		return nil, err
	}
	for _, raw := range documents {
		matchInfo, err := parseYAMLMatchInfo(raw)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		json, err := yaml.YAMLToJSON([]byte(raw))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		resources = append(resources, resource{
			raw:       raw,
			json:      json,
			matchInfo: matchInfo,
		})
	}
	return resources, nil
}

func splitYAMLDocuments(yamlDocumentStream string) ([]string, error) {
	documents := []string{}
	scanner := bufio.NewScanner(strings.NewReader(yamlDocumentStream))
	scanner.Split(splitYAMLDocument)
	for scanner.Scan() {
		documents = append(documents, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, errors.Wrap(err, "error splitting documents")
	}
	return documents, nil
}

const yamlSeparator = "\n---"

// splitYAMLDocument is a bufio.SplitFunc for splitting YAML streams into individual documents.
// this is borrowed from k8s.io/apimachinery/pkg/util/yaml/decoder.go
func splitYAMLDocument(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	sep := len([]byte(yamlSeparator))
	if i := bytes.Index(data, []byte(yamlSeparator)); i >= 0 {
		// We have a potential document terminator
		i += sep
		after := data[i:]
		if len(after) == 0 {
			// we can't read any more characters
			if atEOF {
				return len(data), data[:len(data)-sep], nil
			}
			return 0, nil, nil
		}
		if j := bytes.IndexByte(after, '\n'); j >= 0 {
			return i + j + 1, data[0 : i-sep], nil
		}
		return 0, nil, nil
	}
	// If we're at EOF, we have a final, non-terminated line. Return it.
	if atEOF {
		return len(data), data, nil
	}
	// Request more data.
	return 0, nil, nil
}

// we match resources and patches on their v1 TypeMeta
type matchInfo struct {
	Kind       string `json:"kind,omitempty"`
	APIVersion string `json:"apiVersion,omitempty"`
}

func parseYAMLMatchInfo(raw string) (matchInfo, error) {
	m := matchInfo{}
	if err := yaml.Unmarshal([]byte(raw), &m); err != nil {
		return matchInfo{}, errors.Wrapf(err, "failed to parse type meta for %q", raw)
	}
	return m, nil
}

func matchInfoForConfigJSON6902Patch(patch PatchJSON6902) matchInfo {
	return matchInfo{
		Kind:       patch.Kind,
		APIVersion: groupVersionToAPIVersion(patch.Group, patch.Version),
	}
}

func groupVersionToAPIVersion(group, version string) string {
	if group == "" {
		return version
	}
	return group + "/" + version
}

type mergePatch struct {
	raw       string    // the original raw data
	json      []byte    // the processed data (in JSON form)
	matchInfo matchInfo // for matching resources
}

func parseMergePatches(rawPatches []string) ([]mergePatch, error) {
	patches := []mergePatch{}
	for _, raw := range rawPatches {
		matchInfo, err := parseYAMLMatchInfo(raw)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		json, err := yaml.YAMLToJSON([]byte(raw))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		patches = append(patches, mergePatch{
			raw:       raw,
			json:      json,
			matchInfo: matchInfo,
		})
	}
	return patches, nil
}

type json6902Patch struct {
	raw       string          // raw original contents
	patch     jsonpatch.Patch // processed JSON 6902 patch
	matchInfo matchInfo       // used to match resources
}

func convertJSON6902Patches(patchesJSON6902 []PatchJSON6902) ([]json6902Patch, error) {
	patches := []json6902Patch{}
	for _, configPatch := range patchesJSON6902 {
		patchJSON, err := yaml.YAMLToJSON([]byte(configPatch.Patch))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		patch, err := jsonpatch.DecodePatch(patchJSON)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		patches = append(patches, json6902Patch{
			raw:       configPatch.Patch,
			patch:     patch,
			matchInfo: matchInfoForConfigJSON6902Patch(configPatch),
		})
	}
	return patches, nil
}
