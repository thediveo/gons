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

package gons_test

import (
	"fmt"
	"os"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/thediveo/gons"
	"github.com/thediveo/gons/reexec"
	"github.com/thediveo/lxkns/ops"
	"github.com/thediveo/testbasher"
)

func init() {
	reexec.Register("foo", func() {})
	reexec.Register("enter", func() {
		ns := []string{}
		for _, t := range []string{"user", "mnt", "net"} {
			nsid, _ := ops.NamespacePath("/proc/self/ns/" + t).ID()
			ns = append(ns, fmt.Sprintf("%d", nsid.Ino))
		}
		fmt.Fprintln(os.Stdout, "[", strings.Join(ns, ","), "]")
	})
}

var _ = Describe("gons", func() {

	// Re-execute with an invalid namespace reference.
	It("aborts re-execution for invalid namespace reference", func() {
		Expect(reexec.RunReexecAction(
			"foo",
			reexec.Namespaces([]reexec.Namespace{
				{Type: "net", Path: "/foo"},
			}),
		)).To(MatchError(MatchRegexp(
			`.* ReexecAction.Run: child failed with stderr message ` +
				`".* invalid gons_net reference .*`)))
	})

	// Re-execute and switch into other namespaces especially created for this
	// test.
	It("switches namespaces when re-executing", func() {
		b := testbasher.Basher{}
		defer b.Done()
		b.Script("unshare", `
unshare -Umn $printinfo
`)
		b.Script("printinfo", `
for nst in user mnt net; do
	echo "\"/proc/$$/ns/$nst\""
done
read # wait for Proceed()
`)
		cmd := b.Start("unshare")
		defer cmd.Close()
		var userns, mntns, netns string
		// read the filesystem path references to newly created namespaces.
		cmd.Decode(&userns)
		cmd.Decode(&mntns)
		cmd.Decode(&netns)
		var nsids []uint64
		Expect(reexec.RunReexecAction(
			"enter",
			reexec.Namespaces([]reexec.Namespace{
				{Type: "!user", Path: userns},
				{Type: "!mnt", Path: mntns},
				{Type: "!net", Path: netns},
			}),
			reexec.Result(&nsids),
		)).ToNot(HaveOccurred())
		Expect(nsids).To(Equal([]uint64{
			ID(userns),
			ID(mntns),
			ID(netns),
		}))
	})

	It("converts ns switch errors to text", func() {
		nse := gons.NamespaceSwitchError{}
		Expect(nse.Error()).To(Equal(""))
		var n *gons.NamespaceSwitchError
		Expect(n.Error()).To(Equal("<nil>"))
	})

})

func ID(p string) uint64 {
	id, _ := ops.NamespacePath(p).ID()
	return id.Ino
}
