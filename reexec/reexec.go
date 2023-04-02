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
	"io"
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

// ReexecAction describes a named action to be re-executed in a forked child
// copy of this process, together with its mandatory parameters and options.
type ReexecAction struct {
	ActionName  string      // name of action to run in re-executed child.
	Namespaces  []Namespace // namespaces to switch into before executing action.
	Param       interface{} // optional parameter to be sent to the action.
	Result      interface{} // where to put the action result to.
	Environment []string    // optional environment variables to pass to re-executed child.
}

// ReexecActionOption is an option function configuring some aspect of a
// ReexecAction object. It can be passed to NewReexecAction when creating a
// named action to be re-executed in a forked child copy of our process.
type ReexecActionOption func(*ReexecAction)

// NewReexecAction returns a new ReexecAction object, tailored according to the
// additionally specified options.
func NewReexecAction(actionname string, options ...ReexecActionOption) *ReexecAction {
	a := &ReexecAction{
		ActionName: actionname,
	}
	for _, opt := range options {
		opt(a)
	}
	return a
}

// RunExecAction runs the named action in a forked and re-executed child copy of
// this process with the specified options and returns only after the action in
// the child has finished.
func RunReexecAction(actionname string, options ...ReexecActionOption) error {
	return NewReexecAction(actionname, options...).Run()
}

// Namespaces specifies the namespaces an (re-executed) named action is to be
// run in.
func Namespaces(namespaces []Namespace) ReexecActionOption {
	return func(a *ReexecAction) {
		a.Namespaces = namespaces
	}
}

// Param specifies an (optional) parameter to be sent to the (re-executed) named
// action.
func Param(param interface{}) ReexecActionOption {
	return func(a *ReexecAction) {
		a.Param = param
	}
}

// Result specifies where to place the result received from the (re-executed)
// named action.
func Result(result interface{}) ReexecActionOption {
	return func(a *ReexecAction) {
		a.Result = result
	}
}

// Environment specifies (optional) environment variables passed to the
// (re-executed) named action.
func Environment(environment []string) ReexecActionOption {
	return func(a *ReexecAction) {
		a.Environment = environment
	}
}

// Run restarts the application using reexec and thus as a new child process,
// then immediately executes only the this named action. It optionally passes a
// parameter (as JSON) and/or additional environment variables to the child. The
// output of the child gets deserialized as JSON into the passed result element.
// The call only returns after the child process has terminated.
func (a *ReexecAction) Run() (err error) {
	// Safeguard against applications trying to run more elaborate discoveries
	// and are forgetting to enable the required re-execution of themselves by
	// calling CheckAction() very early in their runtime live.
	if !reexecEnabled {
		if actionname := os.Getenv(magicEnvVar); actionname == "" {
			panic("gons/reexec: ReexecAction.Run: application does not support " +
				"forking and restarting, needs to call reexec.CheckAction() " +
				"first before running discovery")
		}
		panic("gons/reexec: ReexecAction.Run: tried to re-execute in " +
			"already re-executing child process")
	}
	if _, ok := actions[a.ActionName]; !ok {
		panic("gons/reexec: ReexecAction.Run: attempting to re-execute into " +
			"unregistered action \"" + a.ActionName + "\"")
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
	forkchild.Env = append(os.Environ(), a.Environment...)
	// Pass the namespaces the fork/child should switch into via the
	// soon-to-be child's environment. The sequence of the namespaces slice is
	// kept, so that the caller has control of the exact sequence of namespace
	// switches.
	ooorder := []string{} // cSpell:ignore ooorder
	for _, ns := range a.Namespaces {
		ooorder = append(ooorder, ns.Type)
		forkchild.Env = append(forkchild.Env,
			fmt.Sprintf("gons_%s=%s", strings.TrimPrefix(ns.Type, "!"), ns.Path))
	}
	forkchild.Env = append(forkchild.Env, "gons_order="+strings.Join(ooorder, ","))
	// Finally set the action to run on restarting our fork, and then try to
	// start our re-executed fork child...
	forkchild.Env = append(forkchild.Env, magicEnvVar+"="+a.ActionName)
	// If necessary, prepare a JSON encode to send input data to the child
	// process via the child's stdin.
	var encoder *json.Encoder
	if a.Param != nil {
		childin, err := forkchild.StdinPipe()
		if err != nil {
			panic(fmt.Sprintf(
				"gons/reexec: ReexecAction.Run: cannot prepare for restarting my fork, reason: %s",
				err.Error()))
		}
		defer childin.Close()
		encoder = json.NewEncoder(childin)
	}
	// Get the stdout pipe from the child.
	childout, err := forkchild.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf(
			"gons/reexec: ReexecAction.Run: cannot prepare for restarting my fork, reason: %s",
			err.Error()))
	}
	defer childout.Close()
	// Get the stderr pipe from the child and collect any data we might receive.
	// Unfortunately, we can't use the buffer writer directly without further
	// measures as this creates a race condition in those situations where we
	// need to kill the child process: we need to know when the stderr pipe has
	// been closed.
	var childerr bytes.Buffer
	errpipe, err := forkchild.StderrPipe()
	if err != nil {
		panic(fmt.Sprintf(
			"gons/reexec: ReexecAction.Run: cannot prepare for restarting my fork, reason: %s",
			err.Error()))
	}
	errdone := make(chan struct{}, 1)
	go func() {
		defer close(errdone)
		io.Copy(&childerr, errpipe)
	}()
	decoder := json.NewDecoder(childout)
	if err := forkchild.Start(); err != nil {
		panic("gons/reexec: ReexecAction.Run: cannot restart a fork of myself")
	}
	// Sent the optional parameter, if any...
	var encodererr error
	if encoder != nil {
		encodererr = encoder.Encode(a.Param)
	}
	// Decode the result as it flows in. Keep any error for later. Skip this
	// step if we had an encoder error already, as the action won't have got its
	// paremeters correctly.
	var decodererr error
	if encodererr == nil {
		decodererr = decoder.Decode(a.Result)
	}
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
	// Wait for the stderr pipe to properly wind down, so we got all that there
	// is to get.
	<-errdone
	// Any child stderr output takes precedence over decoder errors, as when the
	// child panics, then that is of more importance than any hiccup the result
	// decoder encounters due to the child's problems. However, any encoder
	// error takes it all...
	if encodererr != nil {
		return fmt.Errorf(
			"gons/reexec: ReexecAction.Run: cannot send parameter to child, reason: %w",
			decodererr)
	}
	childhiccup := childerr.String()
	if childhiccup != "" {
		return fmt.Errorf(
			"gons/reexec: ReexecAction.Run: child failed with stderr message %q",
			childhiccup)
	}
	if decodererr != nil {
		return fmt.Errorf(
			"gons/reexec: ReexecAction.Run: cannot decode child result, reason: %w",
			decodererr)
	}
	return err
}

// ForkReexec restarts the application using reexec as a new child process and
// then immediately executes only the specified action (actionname). The output
// of the child gets deserialized as JSON into the passed result element. The
// call returns after the child process has terminated.
//
// Deprecated: use RunReexecAction("foo", Namespaces(n), Result(r)) instead.
func ForkReexec(actionname string, namespaces []Namespace, result interface{}) (err error) {
	return RunReexecAction(
		actionname,
		Namespaces(namespaces),
		Result(result))
}

// ForkReexecEnv restarts the application using reexec as a new child process
// and then immediately executes only the specified action (actionname), passing
// additional environment variables to the child. The output of the child gets
// deserialized as JSON into the passed result element. The call returns after
// the child process has terminated.
//
// Deprecated: use RunReexecAction("foo", Namespaces(n), Environment(env),
// Result(r)) instead.
func ForkReexecEnv(actionname string, namespaces []Namespace, envvars []string, result interface{}) (err error) {
	return RunReexecAction(
		actionname,
		Namespaces(namespaces),
		Environment(envvars),
		Result(result))
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
