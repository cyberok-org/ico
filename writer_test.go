package ico

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"os"
	"testing"
)

func TestEncode(t *testing.T) {
	t.Parallel()
	origfile := "testdata/golang.ico"

	f, err := os.Open("testdata/golang.png")
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	var buf bytes.Buffer
	if err = Encode(&buf, img); err != nil {
		t.Fatal(err)
	}

	f, err = os.Open(origfile)
	if err != nil {
		t.Error(err)
	}
	origICO, err := Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	f.Close()

	newICO, err := Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Error(err)
	}

	inrgba, ok := origICO.(*image.NRGBA)
	if !ok {
		t.Fatal("not nrgba")
	}
	pnrgba, ok := newICO.(*image.NRGBA)
	if !ok {
		t.Fatal("new not nrgba")
	}
	if b, err := fastCompare(inrgba, pnrgba); err != nil || b != 0 {
		t.Fatalf("pix differ %d %v\n", b, err)
	}

}

func TestEncodeWriteErrors(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))

	if err := Encode(&failWriter{failAt: 1}, img); err == nil {
		t.Fatal("expected header write error")
	}
	if err := Encode(&failWriter{failAt: 2}, img); err == nil {
		t.Fatal("expected png write error")
	}
}

type failWriter struct {
	writes int
	failAt int
}

func (w *failWriter) Write(p []byte) (int, error) {
	w.writes++
	if w.writes == w.failAt {
		return 0, errors.New("write failed")
	}
	return len(p), nil
}
