//go:build linux && (amd64 || arm64) && !musl

package zpdf

/*
#cgo linux,amd64 LDFLAGS: ${SRCDIR}/lib/linux_amd64/libzpdf.a -lm -lpthread
#cgo linux,arm64 LDFLAGS: ${SRCDIR}/lib/linux_arm64/libzpdf.a -lm -lpthread
*/
import "C"
