// Reexec support; because the Golang runtime sucks at fork() and switching
// Linux kernel namespaces.

// Copyright 2020 Harald Albrecht.
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

package reexec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/thediveo/gons"
	"github.com/thediveo/gons/reexec/internal/testsupport"
)

// Breaks the vicious cycle of recursive imports which would otherwise raise
// its ugly head: this way, gons/reexec/testing can call RunAction while under
// test, without having to import us. Instead, we and gons/reexec/testing both
// import gons/reexec/internal/testsupport, which in turn doesn't import
// anything which would cause import cycles.
func init() {
	testsupport.RunAction = RunAction
}

// magicEnvVar defines the name of the environment variable which triggers a
// specific registered action to be run when an application using the reexec
// package forks and restarts itself, typically to switch into different
// namespaces.
const magicEnvVar = "gons_reexec_action"

// reexecEnabled enables fork/restarts only for applications which are
// reexec-aware by calling CheckAction() as early as possible in their
// main()s. Applications (indirectly) using reexec and triggering some
// function that needs fork/re-execution, but which have not called
// CheckAction() will panic instead of forking and re-executing themselves.
// This is a safeguard measure to cause havoc by unexpected clone restarts.
var reexecEnabled = false

// CheckAction checks if an application using reexec has been forked and
// re-executed in order to switch namespaces in the clone. If we're in a
// re-execution, then this function won't return, but instead run the
// scheduled reexec functionality. Please do not confuse re-execution with
// royalists and round-heads.
func CheckAction() {
	if RunAction() {
		osExit(0)
	}
}

// For the sake of code coverage ;)
var osExit = os.Exit

// RunAction checks if an application using the gons/reexec package has been
// forked and re-executed as a copy of itself. If this is the case, then the
// action specified for re-execution is run, and true returned. If this isn't
// the case, because this is the parent process and not a re-executed child,
// then no action is run, and false returned instead.
func RunAction() (action bool) {
	// Did we had a problem during reentry...?
	if err := gons.Status(); err != nil {
		panic(err)
	}
	if actionname := os.Getenv(magicEnvVar); actionname != "" {
		// Only run the requested action, and then exit. The caller will never
		// gain back control in this case.
		action, ok := actions[actionname]
		if !ok {
			panic(fmt.Sprintf(
				"unregistered gons/reexec re-execution action %q", actionname))
		}
		action()
		return true
	}
	// Enable fork/re-execution only for the parent process of the application
	// using reexec, but not in the re-executed child.
	reexecEnabled = true
	return
}

// Namespace describes a Linux kernel namespace into which a forked and
// re-executed child process should switch: its type and a path to reference
// it. The type can optionally preceded by a bang "!" which indicates that the
// corresponding path should be opened before any namespace switching occurs;
// without a bang, the path will be opened only right when this namespace
// should be switched. Thus, the path will depend on the current set of
// namespaces, not the initial set when calling ForkReexec().
type Namespace struct {
	Type string // namespace type, such as "net", "mnt", ...
	Path string // path reference to namespace in filesystem.
}

// ForkReexec restarts the application using reexec as a new child process and
// then immediately executes only the specified action (actionname). The
// output of the child gets deserialized as JSON into the passed result
// element. The call returns after the child process has terminated.
func ForkReexec(actionname string, namespaces []Namespace, result interface{}) (err error) {
	return ForkReexecEnv(actionname, namespaces, nil, result)
}

// ForkReexecEnv restarts the application using reexec as a new child process
// and then immediately executes only the specified action (actionname),
// passing additional environment variables to the child. The output of the
// child gets deserialized as JSON into the passed result element. The call
// returns after the child process has terminated.
func ForkReexecEnv(actionname string, namespaces []Namespace, envvars []string, result interface{}) (err error) {
	// Safeguard against applications trying to run more elaborate discoveries
	// and are forgetting to enable the required re-execution of themselves by
	// calling CheckAction() very early in their runtime live.
	if !reexecEnabled {
		if actionname := os.Getenv(magicEnvVar); actionname == "" {
			panic("gons/reexec: ForkReexec: application does not support " +
				"forking and restarting, needs to call reexec.CheckAction() " +
				"first before running discovery")
		}
		panic("gons/reexec: ForkReexec: tried to re-execute in " +
			"already re-executing child process")
	}
	if _, ok := actions[actionname]; !ok {
		panic("gons/reexec: ForkReexec: attempting to re-execute into " +
			"unregistered action \"" + actionname + "\"")
	}
	// If testing has been enabled, then make sure to pass the necessary
	// parameters on to our child processes, as it will (have to) use a
	// TestMain and our "enhanced" gons.reexec.testing.M.
	//
	// When under test, we need to run tests, as otherwise no coverage profile
	// data would be written (if requested by passing an non-empty
	// "-test.coverprofile"), so we make sure to run an empty set of tests;
	// this avoids the same tests getting run multiple times ... and
	// eventually panicking when trying to re-execute again.
	//
	// If coverage propfiling is enabled, then for each child we allocate a
	// separate child coverage profile data file, which we will have to merge
	// later with our main coverage profile of this process.
	testargs := testsupport.TestingArgs()
	// Prepare a fork/re-execution of ourselves, which then switches itself
	// into the required namespace(s) before its Go runtime spins up.
	forkchild := exec.Command("/proc/self/exe", testargs...)
	forkchild.Env = append(os.Environ(), envvars...)
	// Pass the namespaces the fork/child should switch into via the
	// soon-to-be child's environment. The sequence of the namespaces slice is
	// kept, so that the caller has control of the exact sequence of namespace
	// switches.
	ooorder := []string{} // cSpell:ignore ooorder
	for _, ns := range namespaces {
		ooorder = append(ooorder, ns.Type)
		forkchild.Env = append(forkchild.Env,
			fmt.Sprintf("gons_%s=%s", strings.TrimPrefix(ns.Type, "!"), ns.Path))
	}
	forkchild.Env = append(forkchild.Env, "gons_order="+strings.Join(ooorder, ","))
	// Finally set the action to run on restarting our fork, and then try to
	// start our re-executed fork child...
	forkchild.Env = append(forkchild.Env, magicEnvVar+"="+actionname)
	childout, err := forkchild.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf("gons/reexec: ForkReexec: cannot prepare for restart my fork, %s", err.Error()))
	}
	defer childout.Close()
	var childerr bytes.Buffer
	forkchild.Stderr = &childerr
	decoder := json.NewDecoder(childout)
	if err := forkchild.Start(); err != nil {
		panic(fmt.Sprintf("gons/reexec: ForkReexec: cannot restart a fork of myself"))
	}
	// Decode the result as it flows in. Keep any error for later...
	decodererr := decoder.Decode(result)
	// Either wait for the child to automatically terminate within a short
	// grace period after we deserialized its result output, or kill it the
	// hard way if it can't terminate in time.
	done := make(chan error, 1)
	go func() {
		done <- forkchild.Wait()
	}()
	select {
	case err = <-done:
	case <-time.After(1 * time.Second):
		_ = forkchild.Process.Kill()
	}
	// Any child stderr output takes precedence over decoder errors, as when
	// the child panics, then that is of more importance than any hiccup the
	// result decoder encounters due to the child's problems.
	childhiccup := childerr.String()
	if childhiccup != "" {
		return fmt.Errorf(
			"gons/reexec: ForkReexec: child failed with stderr message: %q",
			childhiccup)
	}
	if decodererr != nil {
		return fmt.Errorf(
			"gons/reexec: ForkReexec: cannot decode child result, %q",
			decodererr.Error())
	}
	return err
}

// Action is a function that is run on demand during re-execution of a forked
// child.
type Action func()

// actions maps re-execution topics (names) to action functions to execute on
// a scheduled re-execution.
var actions = map[string]Action{}

// Register registers a Action function with a name so it can be
// triggered during ForkReexec(name, ...). The registration panics if the same
// Action name is registered more than once, regardless of whether with the
// same Action or different ones.
func Register(name string, action Action) {
	if _, ok := actions[name]; ok {
		panic(fmt.Sprintf(
			"gons/reexec: registerAction: re-execution action %q already registered",
			name))
	}
	actions[name] = action
}
