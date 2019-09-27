// +build linux

package gons

/*
extern void gonamespaces(void);
void __attribute__((constructor)) init(void) {
	gonamespaces();
}
*/
import "C"
