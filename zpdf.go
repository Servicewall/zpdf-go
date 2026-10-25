// Package zpdf provides PDF text extraction backed by the zpdf Zig library,
// plus optional page image extraction for OCR.
//
// The static library for each supported platform is bundled with this module;
// callers do not need cgo, a C toolchain, or any system PDF library.
//
// Supported platforms: darwin/arm64, linux/amd64, linux/arm64.
package zpdf

/*
#cgo CFLAGS: -I${SRCDIR}/include
#include <stdlib.h>
#include "zpdf.h"
*/
import "C"

import (
	"bytes"
	"errors"
	"runtime"
	"unsafe"
)

const abiVersion = 2

// Document is an open PDF. It is not safe for concurrent use.
type Document struct {
	p *C.zpdf_document
}

// PageInfo describes a page's media box and rotation.
type PageInfo struct {
	Width    float64
	Height   float64
	Rotation int
}

// ImageFormat is the encoding of an extracted page image.
type ImageFormat int

const (
	ImageRGB24 ImageFormat = iota
	ImageGray8
	ImageJPEG
	ImageCCITT
	ImageJPEG2000
)

// Image is one image extracted from a page for OCR.
type Image struct {
	Page       int
	Rect       [4]float64 // x0, y0, x1, y1 in page coordinates
	Width      int
	Height     int
	Format     ImageFormat
	BPC        int
	ColorSpace int
	CCITTK     int
	BlackIs1   bool
	Data       []byte
}

// Error is an error returned by the zpdf C library.
type Error int

const (
	ErrInvalidHandle Error = iota + 1
	ErrPageNotFound
	ErrExtractionFailed
	ErrOutOfMemory
)

func (e Error) Error() string {
	switch e {
	case ErrInvalidHandle:
		return "zpdf: invalid document handle"
	case ErrPageNotFound:
		return "zpdf: page not found"
	case ErrExtractionFailed:
		return "zpdf: text extraction failed"
	case ErrOutOfMemory:
		return "zpdf: out of memory"
	default:
		return "zpdf: unknown error"
	}
}

var (
	errABIMismatch = errors.New("zpdf: C library ABI mismatch")
	// ErrInvalidPDF is returned when the input is not a parseable PDF.
	ErrInvalidPDF = errors.New("zpdf: invalid PDF")
)

func init() {
	if v := uint32(C.zpdf_abi_version()); v != abiVersion {
		panic(errABIMismatch)
	}
}

func lastError() error {
	if code := int(C.zpdf_last_error()); code != 0 {
		return Error(code)
	}
	return ErrExtractionFailed
}

// Open parses data. The document keeps its own copy, so data can be released
// as soon as Open returns.
func Open(data []byte) (*Document, error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, ErrInvalidPDF
	}
	p := C.zpdf_open_memory_copy((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if p == nil {
		return nil, lastError()
	}
	d := &Document{p: p}
	if d.PageCount() <= 0 {
		d.Close()
		return nil, ErrInvalidPDF
	}
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// OpenBorrowed parses data without copying. data must stay alive and unchanged
// until Close. Prefer Open unless you have measured the copy cost.
func OpenBorrowed(data []byte) (*Document, error) {
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		return nil, ErrInvalidPDF
	}
	p := C.zpdf_open_memory((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)))
	if p == nil {
		return nil, lastError()
	}
	d := &Document{p: p}
	if d.PageCount() <= 0 {
		d.Close()
		return nil, ErrInvalidPDF
	}
	runtime.SetFinalizer(d, (*Document).Close)
	return d, nil
}

// Close releases the document. It is safe to call more than once.
func (d *Document) Close() {
	if d == nil || d.p == nil {
		return
	}
	C.zpdf_close(d.p)
	d.p = nil
	runtime.SetFinalizer(d, nil)
}

// PageCount returns the number of pages.
func (d *Document) PageCount() int {
	if d == nil || d.p == nil {
		return 0
	}
	return int(C.zpdf_page_count(d.p))
}

// IsEncrypted reports whether the PDF is encrypted. Text extraction of an
// encrypted document is not supported.
func (d *Document) IsEncrypted() bool {
	if d == nil || d.p == nil {
		return false
	}
	return bool(C.zpdf_is_encrypted(d.p))
}

// AllText extracts all pages in content-stream order (the closest match to
// `pdftotext -raw`, with line repair).
func (d *Document) AllText() (string, error) {
	return d.extract(func(n *C.size_t) *C.uint8_t {
		return C.zpdf_extract_all_fast(d.p, n)
	})
}

// AllTextReadingOrder extracts all pages using the structure tree when the PDF
// is tagged, and geometric ordering otherwise.
func (d *Document) AllTextReadingOrder() (string, error) {
	return d.extract(func(n *C.size_t) *C.uint8_t {
		return C.zpdf_extract_all(d.p, n)
	})
}

// PageText extracts a single page in content-stream order.
func (d *Document) PageText(page int) (string, error) {
	if page < 0 || page >= d.PageCount() {
		return "", ErrPageNotFound
	}
	return d.extract(func(n *C.size_t) *C.uint8_t {
		return C.zpdf_extract_page(d.p, C.int(page), n)
	})
}

func (d *Document) extract(fn func(*C.size_t) *C.uint8_t) (string, error) {
	if d == nil || d.p == nil {
		return "", ErrInvalidHandle
	}
	var n C.size_t
	ptr := fn(&n)
	if ptr == nil {
		return "", lastError()
	}
	defer C.zpdf_free_buffer(ptr, n)
	return C.GoStringN((*C.char)(unsafe.Pointer(ptr)), C.int(n)), nil
}

// PageImages returns the images on a page (opt-in; use it only for OCR).
func (d *Document) PageImages(page int) ([]Image, error) {
	if d == nil || d.p == nil {
		return nil, ErrInvalidHandle
	}
	if page < 0 || page >= d.PageCount() {
		return nil, ErrPageNotFound
	}
	var out *C.ZpdfPageImage
	var count C.size_t
	if code := C.zpdf_extract_page_images(d.p, C.int(page), &out, &count); code != 0 {
		return nil, lastError()
	}
	defer C.zpdf_free_page_images(out, count)
	if count == 0 {
		return nil, nil
	}
	arr := unsafe.Slice(out, int(count))
	imgs := make([]Image, len(arr))
	for i, ci := range arr {
		var data []byte
		if ci.data != nil && ci.len > 0 {
			data = C.GoBytes(unsafe.Pointer(ci.data), C.int(ci.len))
		}
		imgs[i] = Image{
			Page:       int(ci.page),
			Rect:       [4]float64{float64(ci.x0), float64(ci.y0), float64(ci.x1), float64(ci.y1)},
			Width:      int(ci.width),
			Height:     int(ci.height),
			Format:     ImageFormat(ci.format),
			BPC:        int(ci.bpc),
			ColorSpace: int(ci.colorspace),
			CCITTK:     int(ci.ccitt_k),
			BlackIs1:   bool(ci.black_is_1),
			Data:       data,
		}
	}
	return imgs, nil
}

// PageInfo returns the page size and rotation.
func (d *Document) PageInfo(page int) (PageInfo, error) {
	if d == nil || d.p == nil {
		return PageInfo{}, ErrInvalidHandle
	}
	if page < 0 || page >= d.PageCount() {
		return PageInfo{}, ErrPageNotFound
	}
	var w, h C.double
	var rot C.int
	if code := C.zpdf_get_page_info(d.p, C.int(page), &w, &h, &rot); code != 0 {
		return PageInfo{}, lastError()
	}
	return PageInfo{Width: float64(w), Height: float64(h), Rotation: int(rot)}, nil
}
