// Copyright 2019 Harald Albrecht.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build linux
// +build linux

package gons

/*
extern void gonamespaces(void);
extern char *gonsmsg;
void __attribute__((constructor)) init(void) {
	gonamespaces();
}
*/
import "C"

// NamespaceSwitchError reports unsuccessful namespace switching during
// startup.
type NamespaceSwitchError struct {
	details string
}

// Error returns a description of the failure causing the initial namespace
// switching to abort.
func (e *NamespaceSwitchError) Error() string {
	if e == nil {
		return "<nil>"
	}
	return e.details
}

// Status returns nil if there were no problems switching namespaces during
// initial startup; otherwise, it returns a NamespaceSwitchError with a detail
// description.
func Status() error {
	if C.gonsmsg == nil {
		return nil
	}
	return &NamespaceSwitchError{
		details: C.GoString(C.gonsmsg),
	}
}
