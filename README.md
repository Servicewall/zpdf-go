# zpdf-go

Go bindings for [zpdf](https://github.com/donge/zpdf), a fast PDF text
extraction library written in Zig.

The static libraries are bundled per platform and cgo is an implementation
detail: callers only need Go.

```bash
go get github.com/Servicewall/zpdf-go
```

## Usage

```go
import "github.com/Servicewall/zpdf-go"

doc, err := zpdf.Open(data) // copies data; safe to release it afterwards
if err != nil {
    return err
}
defer doc.Close()

if doc.IsEncrypted() {
    return errors.New("encrypted PDF")
}

text, err := doc.AllText() // stream order, like `pdftotext -raw`
```

Per page:

```go
for i := 0; i < doc.PageCount(); i++ {
    text, err := doc.PageText(i)
    // ...
}
```

Reading order (structure tree when tagged, geometric fallback):

```go
text, err := doc.AllTextReadingOrder()
```

Page geometry:

```go
info, _ := doc.PageInfo(0) // Width, Height, Rotation (0/90/180/270)
```

## API

| Function | Description |
|---|---|
| `Open([]byte) (*Document, error)` | Parse a PDF, keeping a copy of the input |
| `OpenBorrowed([]byte) (*Document, error)` | Zero-copy; the slice must stay alive until `Close` |
| `(*Document) Close()` | Release resources (idempotent) |
| `(*Document) PageCount() int` | Number of pages |
| `(*Document) IsEncrypted() bool` | Encrypted PDFs are not decrypted |
| `(*Document) AllText() (string, error)` | All pages, content-stream order |
| `(*Document) AllTextReadingOrder() (string, error)` | All pages, reading order |
| `(*Document) PageText(page int) (string, error)` | One page, content-stream order |
| `(*Document) PageInfo(page int) (PageInfo, error)` | MediaBox size and rotation |

Errors: `ErrInvalidPDF`, `ErrUnsupported` (platform), `ErrPageNotFound`,
`ErrExtractionFailed`, `ErrOutOfMemory`.

## Performance

Measured on 50 real PDFs / 773 pages (38 with a text layer), in-process, warm
cache:

| engine | total | median ms/page | mean coverage vs poppler |
|---|---:|---:|---:|
| **zpdf** | **0.14 s** | **0.10** | **0.993** |
| poppler | 1.00 s | 0.57 | 1.000 |
| PDFium | 1.07 s | 0.71 | 0.995 |
| MuPDF | 0.84 s | 0.67 | 0.999 |

See the [zpdf README](https://github.com/donge/zpdf) for the full comparison
and the list of fixed defects (xref recovery, Form XObject fonts, CJK Unicode,
line reconstruction).

## Platform support

| GOOS / GOARCH | status |
|---|---|
| linux/amd64 | supported (glibc 2.28+) |
| linux/arm64 | supported (glibc 2.28+) |
| darwin/arm64 | supported (macOS 13+) |
| darwin/amd64 | supported (macOS 13+) |
| others | `ErrUnsupported` |

The libraries are built from zpdf `v0.3.1` (Zig 0.17.0):

```bash
# in donge/zpdf
zig build capi -Doptimize=ReleaseFast -Dtarget=aarch64-macos.13.0 --prefix dist/darwin-arm64
zig build capi -Doptimize=ReleaseFast -Dtarget=x86_64-macos.13.0 --prefix dist/darwin-amd64
zig build capi -Doptimize=ReleaseFast -Dtarget=x86_64-linux-gnu.2.28 --prefix dist/linux-amd64
zig build capi -Doptimize=ReleaseFast -Dtarget=aarch64-linux-gnu.2.28 --prefix dist/linux-arm64
```

On macOS the archive must be re-archived with `libtool -static` so Apple's
linker accepts the member alignment:

```bash
cd /tmp && mkdir ar && cd ar
ar x <dist>/lib/libzpdf_c.a && chmod 644 *.o
libtool -static -o libzpdf.a *.o
```

## Testing

```bash
go test ./...
```

## License

zpdf is CC0-1.0; this binding is MIT.
