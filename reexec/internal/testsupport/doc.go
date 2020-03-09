/*

Package testsupport is an internal package designed to break import cycles
between gons/reexec and gons/reexec/testing: it allows passing information
about coverage profile data files and their locations forth and back between
the two packages when an application using gons/reexec is under test. And it
allows invoking gons/reexec's CheckAction() for triggering a registered action
during re-execution.

*/
package testsupport
