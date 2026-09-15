//go:build cgo && linux

package native

/*
#cgo CFLAGS: -O2 -I${SRCDIR}/../../native/c
#cgo LDFLAGS: ${SRCDIR}/../../native/rust/target/release/libcalculator_rust.a -ldl -lpthread -lm
#include "calculator.h"
int64_t sub(int64_t a, int64_t b);
*/
import "C"

type ffi struct{}

func New() (Calculator, error)   { return ffi{}, nil }
func (ffi) Add(a, b int64) int64 { return int64(C.add(C.int64_t(a), C.int64_t(b))) }
func (ffi) Sub(a, b int64) int64 { return int64(C.sub(C.int64_t(a), C.int64_t(b))) }
