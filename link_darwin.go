//go:build darwin && (arm64 || amd64)

package zpdf

/*
#cgo darwin,arm64 LDFLAGS: ${SRCDIR}/lib/darwin_arm64/libzpdf.a
#cgo darwin,amd64 LDFLAGS: ${SRCDIR}/lib/darwin_amd64/libzpdf.a
*/
import "C"
