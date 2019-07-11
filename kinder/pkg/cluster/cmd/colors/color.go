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
Package colors implement utilities for colorizing prompt, command and args printed to screen before execution.
*/
package colors

import (
	"fmt"
	"os"
	"strings"
)

func colorOn() bool {
	return strings.ToLower(os.Getenv("KINDER_COLORS")) == "on"
}

// Prompt returns a colorized version of the command prompt
func Prompt(hostname string) string {
	if !colorOn() {
		return hostname
	}
	return fmt.Sprintf("\033[1;48;5;19m%s\033[0m", hostname)
}

// Command returns a colorized version of the command string, including also its arguments
func Command(s string) string {
	if !colorOn() {
		return s
	}
	return fmt.Sprintf("\033[1;48;5;33m%s\033[0m", s)
}

// Info return a colorized version of a generic message
func Info(s string) string {
	if !colorOn() {
		return s
	}
	return fmt.Sprintf("\033[1;48;5;34m%s\033[0m", s)
}
