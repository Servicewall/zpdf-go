package zpdf

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

// makePDF builds a minimal, valid one-page PDF.
func makePDF(text string) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	objs := make([]string, 5)
	objs[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	objs[1] = "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"
	objs[2] = "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R " +
		"/Resources << /Font << /F1 5 0 R >> >> >>"
	content := "BT\n/F1 12 Tf\n72 700 Td\n(" + text + ") Tj\nET\n"
	objs[3] = fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)
	objs[4] = "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica /Encoding /WinAnsiEncoding >>"

	offsets := make([]int, len(objs))
	for i, body := range objs {
		offsets[i] = b.Len()
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objs)+1)
	b.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func TestExtract(t *testing.T) {
	data := makePDF("Hello zpdf")
	doc, err := Open(data)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	if got := doc.PageCount(); got != 1 {
		t.Fatalf("PageCount = %d, want 1", got)
	}
	if doc.IsEncrypted() {
		t.Fatal("IsEncrypted = true")
	}
	text, err := doc.AllText()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(text, "Hello zpdf") {
		t.Fatalf("AllText = %q", text)
	}
	page, err := doc.PageText(0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(page, "Hello zpdf") {
		t.Fatalf("PageText = %q", page)
	}
	info, err := doc.PageInfo(0)
	if err != nil {
		t.Fatal(err)
	}
	if info.Width != 612 || info.Height != 792 {
		t.Fatalf("PageInfo = %+v", info)
	}
}

func TestOpenBorrowed(t *testing.T) {
	data := makePDF("Borrowed")
	doc, err := OpenBorrowed(data)
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	text, err := doc.AllText()
	if err != nil || !strings.Contains(text, "Borrowed") {
		t.Fatalf("AllText = %q, err = %v", text, err)
	}
}

func TestErrors(t *testing.T) {
	if _, err := Open(nil); err == nil {
		t.Fatal("Open(nil) = nil error")
	}
	if _, err := Open([]byte("not a pdf")); err == nil {
		t.Fatal("Open(garbage) = nil error")
	}
	doc, err := Open(makePDF("x"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := doc.PageText(5); err != ErrPageNotFound {
		t.Fatalf("PageText(5) err = %v", err)
	}
	doc.Close()
	doc.Close() // idempotent
}

// makeImagePDF builds a one-page PDF with a single image XObject.
func makeImagePDF(w, h int, filter string, data []byte) []byte {
	var b bytes.Buffer
	b.WriteString("%PDF-1.4\n%\xE2\xE3\xCF\xD3\n")
	objs := make([]string, 5)
	objs[0] = "<< /Type /Catalog /Pages 2 0 R >>"
	objs[1] = "<< /Type /Pages /Kids [3 0 R] /Count 1 >>"
	objs[2] = "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Contents 4 0 R " +
		"/Resources << /XObject << /Im1 5 0 R >> >> >>"
	content := "q 100 0 0 100 10 10 cm /Im1 Do Q"
	objs[3] = fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content)
	f := ""
	if filter != "" {
		f = " /Filter " + filter
	}
	objs[4] = fmt.Sprintf("<< /Type /XObject /Subtype /Image /Width %d /Height %d "+
		"/ColorSpace /DeviceRGB /BitsPerComponent 8%s /Length %d >>\nstream\n", w, h, f, len(data))

	offsets := make([]int, len(objs))
	for i, body := range objs {
		offsets[i] = b.Len()
		if i == 4 {
			fmt.Fprintf(&b, "%d 0 obj\n", i+1)
			b.WriteString(body)
			b.Write(data)
			b.WriteString("\nendstream\nendobj\n")
			continue
		}
		fmt.Fprintf(&b, "%d 0 obj\n%s\nendobj\n", i+1, body)
	}
	xref := b.Len()
	fmt.Fprintf(&b, "xref\n0 %d\n", len(objs)+1)
	b.WriteString("0000000000 65535 f \n")
	for _, off := range offsets {
		fmt.Fprintf(&b, "%010d 00000 n \n", off)
	}
	fmt.Fprintf(&b, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref)
	return b.Bytes()
}

func TestPageImagesRGB(t *testing.T) {
	const w, h = 128, 96
	pix := make([]byte, w*h*3)
	for i := range pix {
		pix[i] = byte(i % 251)
	}
	doc, err := Open(makeImagePDF(w, h, "", pix))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	imgs, err := doc.PageImages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 1 {
		t.Fatalf("images = %d, want 1", len(imgs))
	}
	img := imgs[0]
	if img.Format != ImageRGB24 || img.Width != w || img.Height != h {
		t.Fatalf("image = %+v", img)
	}
	if !bytes.Equal(img.Data, pix) {
		t.Fatal("pixel data mismatch")
	}
}

func TestPageImagesJPEGPassthrough(t *testing.T) {
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 1, 2, 3, 0xFF, 0xD9}
	doc, err := Open(makeImagePDF(200, 100, "/DCTDecode", jpeg))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()

	imgs, err := doc.PageImages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 1 {
		t.Fatalf("images = %d, want 1", len(imgs))
	}
	if imgs[0].Format != ImageJPEG || !bytes.Equal(imgs[0].Data, jpeg) {
		t.Fatalf("image = %+v", imgs[0])
	}
}

func TestPageImagesNone(t *testing.T) {
	doc, err := Open(makePDF("no images here"))
	if err != nil {
		t.Fatal(err)
	}
	defer doc.Close()
	imgs, err := doc.PageImages(0)
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 0 {
		t.Fatalf("images = %d, want 0", len(imgs))
	}
}
