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

package testing

import (
	"bufio"
	"os"
	"strings"
)

// pritiPratel runs function f and passes only harmless error output destined
// to os.Stderr really on to os.Stderr. All dangerous talk about unwanted
// error truths will be sent into early retirement. Such as testing-related
// messages, which we silently drop. This is necessary as some applications
// using gons/reexec expect the re-executed child to return their results via
// stdout without any stderr output, so we don't want Golang's testing output
// to interfere here.
func pritiPratel(f func()) {
	realStderr := os.Stderr
	// Unfortunately, we cannot make use of the in-memory io.Pipe()s here, as
	// os.Stderr is a *os.File, so it needs to have a file descriptor. In
	// consequence, that leaves us with the sole option of a "real" pipe.
	reader, writer, err := os.Pipe()
	if err != nil {
		panic("gons/reexec/testing: cannot create filtering pipe: " + err.Error())
	}
	os.Stderr = writer
	defer func() {
		os.Stderr = realStderr
		reader.Close()
		writer.Close()
	}()
	// The stdout filter for filtering out unwanted Golang testing messages.
	// It runs as a separate Go routine, which only terminates on (real) read
	// errors.
	done := make(chan struct{})
	go func() {
		r := bufio.NewReaderSize(reader, 1024)
		for {
			// Assume that we're starting with a new line here, so sort out
			// any output beginning with "coverage:" or "testing:".
			line, isprefix, err := r.ReadLine()
			if err != nil {
				// bufio.Reader.ReadLine() either returns a non-nil line or it
				// returns an error, never both.
				close(done)
				return
			}
			tobehidden := false
			for _, h := range hide {
				if strings.HasPrefix(string(line), h) {
					// This line of output should be dropped. In case this
					// line is longer than the buffer, drop until the end of
					// line chunk by chunk.
					for isprefix {
						_, isprefix, err = r.ReadLine()
						if err != nil {
							close(done)
							return
						}
					}
					tobehidden = true
					break
				}
			}
			if tobehidden {
				continue
			}
			// It's output we should better pass on...
			if _, err := realStderr.Write(line); err != nil {
				close(done)
				return
			}
			for isprefix {
				line, isprefix, err = r.ReadLine()
				if err != nil {
					close(done)
					return
				}
				if _, err := realStderr.Write(line); err != nil {
					close(done)
					return
				}
			}
			// Handle the case where we have read the final line before EOF,
			// which doesn't end in \n: in this situation, we must not append
			// any \n to the output.
			if err := r.UnreadByte(); err != nil {
				close(done)
				return
			}
			if b, err := r.ReadByte(); err == nil && b == '\n' {
				if _, err := realStderr.Write([]byte{'\n'}); err != nil {
					close(done)
					return
				}
			}
		}
		// unreachable
	}()
	// Now run the desired function f() while scanning its output for unwanted
	// rogue lies which we need to suppress. Definitely a Johnson function
	// here.
	f()
	writer.Close()
	<-done
}

var hide = []string{
	"coverage:",
	"testing:",
}
