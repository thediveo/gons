/*
Package testing supports code coverage profiling of applications using the
gons/reexec package.

This is ugly stuff, because it has Golang's testing package to work in the
context of test process re-execution. As testing only writes coverage profile
data at the end of testing.M.Run(), we need to run testing.M.Run() also on
re-execution. However, without further measures the resulting coverage profile
data file of the re-executed child would be overwritten by the parent when it
finishes its own testing.M.Run(). So we need to run each re-executed child on
its own coverage profile data file. After testing.M.Run() has finished, we
then merge the child coverage profile data into the parent's coverage profile
data file.
*/
package testing
