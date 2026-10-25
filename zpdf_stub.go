//go:build !((darwin && arm64) || (linux && (amd64 || arm64)))

package zpdf

import "errors"

var (
	// ErrUnsupported is returned on platforms without a bundled zpdf library.
	ErrUnsupported = errors.New("zpdf: unsupported platform")
	// ErrInvalidPDF is returned when the input is not a parseable PDF.
	ErrInvalidPDF = errors.New("zpdf: invalid PDF")
)

// Document is an open PDF.
type Document struct{}

// PageInfo describes a page's media box and rotation.
type PageInfo struct {
	Width    float64
	Height   float64
	Rotation int
}

// Error is an error returned by the zpdf C library.
type Error int

const (
	ErrInvalidHandle Error = iota + 1
	ErrPageNotFound
	ErrExtractionFailed
	ErrOutOfMemory
)

func (e Error) Error() string { return "zpdf: error" }

func Open([]byte) (*Document, error)          { return nil, ErrUnsupported }
func OpenBorrowed([]byte) (*Document, error)  { return nil, ErrUnsupported }
func (d *Document) Close()                    {}
func (d *Document) PageCount() int            { return 0 }
func (d *Document) IsEncrypted() bool         { return false }
func (d *Document) AllText() (string, error)  { return "", ErrUnsupported }
func (d *Document) AllTextReadingOrder() (string, error) { return "", ErrUnsupported }
func (d *Document) PageText(int) (string, error)         { return "", ErrUnsupported }
func (d *Document) PageInfo(int) (PageInfo, error)       { return PageInfo{}, ErrUnsupported }
