//go:build cgo && linux

#if defined(__GNUC__)
#pragma GCC optimize ("wrapv")
#endif
#include "../../native/c/calculator.c"
