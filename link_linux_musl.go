//go:build linux && (amd64 || arm64) && musl

package zpdf

/*
#cgo linux,amd64 LDFLAGS: ${SRCDIR}/lib/linux_amd64_musl/libzpdf.a -lm
#cgo linux,arm64 LDFLAGS: ${SRCDIR}/lib/linux_arm64_musl/libzpdf.a -lm
*/
import "C"
