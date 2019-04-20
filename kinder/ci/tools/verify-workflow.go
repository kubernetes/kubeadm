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

package main

import (
	"fmt"
	"os"

	ktestworkflow "k8s.io/kubeadm/kinder/pkg/test/workflow"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("error: missing file argument")
		fmt.Println("usage: verify-workflow some-file.yaml")
		os.Exit(1)
	}
	file := os.Args[1]
	fmt.Printf("Verifying %s...", file)
	_, err := ktestworkflow.NewWorkflow(file)
	if err != nil {
		fmt.Printf("FAILED\n%v\n", err)
		os.Exit(1)
	}
	fmt.Println("OK")
}
