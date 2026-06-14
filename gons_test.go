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
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/thediveo/lxkns/ops"
	"github.com/thediveo/spacetest/spacer"

	"github.com/thediveo/gons"
	"github.com/thediveo/gons/reexec"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func init() {
	reexec.Register("foo", func() {})
	reexec.Register("enter", func() {
		ns := []string{}
		for _, t := range []string{"user", "mnt", "net"} {
			nsid, _ := ops.NamespacePath("/proc/self/ns/" + t).ID()
			ns = append(ns, fmt.Sprintf("%d", nsid.Ino))
		}
		_, _ = fmt.Fprintln(os.Stdout, "[", strings.Join(ns, ","), "]")
	})
}

var _ = AfterSuite(func() {
	gexec.CleanupBuildArtifacts()
})

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
	It("switches namespaces when re-executing", func(ctx context.Context) {
		nsref := func(fd int) string {
			return fmt.Sprintf("/proc/%d/fd/%d", os.Getpid(), fd)
		}

		spc := spacer.New(ctx, spacer.WithOut(GinkgoWriter), spacer.WithErr(GinkgoWriter))
		DeferCleanup(spc.Close)
		subspc, usernsfd := spc.NewTransientUser()
		DeferCleanup(subspc.Close)
		userns := nsref(usernsfd)

		morens := subspc.Rooms(false, false, true, true, false, false)
		mntns := nsref(morens.Mnt)
		netns := nsref(morens.Net)

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
