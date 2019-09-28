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
// initial startup; otherwise, it returns a NamespaceSwitchError with details.
func Status() error {
	if C.gonsmsg == nil {
		return nil
	}
	return &NamespaceSwitchError{
		details: C.GoString(C.gonsmsg),
	}
}
