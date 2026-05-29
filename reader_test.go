package ico

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
)

func sqDiffUInt8(x, y uint8) uint64 {
	d := uint64(x) - uint64(y)
	return d * d
}
func fastCompare(img1, img2 *image.NRGBA) (int64, error) {
	if img1.Bounds() != img2.Bounds() {
		return 0, fmt.Errorf("image bounds not equal: %+v, %+v", img1.Bounds(), img2.Bounds())
	}

	accumError := int64(0)

	for i := 0; i < len(img1.Pix); i++ {
		accumError += int64(sqDiffUInt8(img1.Pix[i], img2.Pix[i]))
	}

	return int64(math.Sqrt(float64(accumError))), nil
}

func TestDecodeConfig(t *testing.T) {
	t.Parallel()
	file := "testdata/golang.ico"
	copyFile := "testdata/golang.png"
	reader, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	icoImage, err := DecodeConfig(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}
	reader, err = os.Open(copyFile)
	if err != nil {
		t.Fatal(err)
	}
	pngImage, err := png.DecodeConfig(reader)
	reader.Close()
	if err != nil {
		t.Fatal(err)
	}

	if icoImage != pngImage {
		t.Errorf("%v - %v", icoImage, pngImage)
	}

}

func TestDecode(t *testing.T) {
	t.Parallel()
	file := "testdata/golang.ico"
	copyFile := "testdata/golang.png"
	reader, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	icoImage, err := Decode(reader)
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()

	reader, err = os.Open(copyFile)
	if err != nil {
		t.Fatal(err)
	}
	pngImage, err := png.Decode(reader)
	if err != nil {
		t.Fatal(err)
	}
	reader.Close()

	if icoImage == nil || !icoImage.Bounds().Eq(pngImage.Bounds()) {
		t.Fatal("bounds differ")
	}
	inrgba, ok := icoImage.(*image.NRGBA)
	if !ok {
		t.Fatal("not nrgba")
	}
	pnrgba, ok := pngImage.(*image.NRGBA)
	if !ok {
		t.Fatal("png not nrgba")
	}

	if b, err := fastCompare(inrgba, pnrgba); err != nil || b > 700 {
		t.Fatalf("pix differ %d %v\n", b, err)
	}
}

func TestDecodeAllPNG(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	src := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	src.SetNRGBA(0, 0, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(1, 0, color.NRGBA{G: 255, A: 128})
	src.SetNRGBA(0, 1, color.NRGBA{B: 255, A: 64})
	src.SetNRGBA(1, 1, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	if err := Encode(&buf, src); err != nil {
		t.Fatal(err)
	}

	images, err := DecodeAll(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("expected one image, got %d", len(images))
	}
	if !images[0].Bounds().Eq(src.Bounds()) {
		t.Fatalf("bounds differ: %v - %v", images[0].Bounds(), src.Bounds())
	}
}

func TestGetFirstImageEmpty(t *testing.T) {
	t.Parallel()

	if img := getFirstImage([]image.Image{nil}); img != nil {
		t.Fatalf("expected nil image, got %v", img)
	}
}

func TestDecodeBMP32(t *testing.T) {
	t.Parallel()

	images, err := DecodeAll(bytes.NewReader(buildBMP32ICO()))
	if err != nil {
		t.Fatal(err)
	}
	if len(images) != 1 {
		t.Fatalf("expected one image, got %d", len(images))
	}
	if !images[0].Bounds().Eq(image.Rect(0, 0, 2, 2)) {
		t.Fatalf("unexpected bounds: %v", images[0].Bounds())
	}
	if _, _, _, a := images[0].At(1, 0).RGBA(); a != 0 {
		t.Fatalf("expected transparent pixel alpha, got %d", a)
	}
}

func TestDecodeBMPWithMask(t *testing.T) {
	t.Parallel()

	img, err := Decode(bytes.NewReader(buildBMP24ICOWithMask()))
	if err != nil {
		t.Fatal(err)
	}
	if !img.Bounds().Eq(image.Rect(0, 0, 2, 2)) {
		t.Fatalf("unexpected bounds: %v", img.Bounds())
	}
	if _, _, _, a := img.At(1, 0).RGBA(); a != 0 {
		t.Fatalf("expected masked pixel alpha, got %d", a)
	}
	if _, _, _, a := img.At(0, 0).RGBA(); a == 0 {
		t.Fatal("expected unmasked pixel alpha")
	}
}

func TestDecodeConfigBMP(t *testing.T) {
	t.Parallel()

	cfg, err := DecodeConfig(bytes.NewReader(buildBMP32ICO()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 2 || cfg.Height != 2 {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestDecodeConfigErrors(t *testing.T) {
	t.Parallel()

	t.Run("corrupt header", func(t *testing.T) {
		t.Parallel()
		if _, err := DecodeConfig(bytes.NewReader([]byte{0, 0, 2, 0, 0, 0})); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("short entry", func(t *testing.T) {
		t.Parallel()
		if _, err := DecodeConfig(bytes.NewReader([]byte{0, 0, 1, 0, 1, 0})); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("bmp header error", func(t *testing.T) {
		t.Parallel()
		dib := makeBMPDIB(maxSize+1, 1, 32, nil, nil)
		data := buildICO(dib, direntry{Width: 1, Height: 1, Plane: 1, Bits: 32, Size: uint32(len(dib)), Offset: 22})
		if _, err := DecodeConfig(bytes.NewReader(data)); err == nil {
			t.Fatal("expected error")
		}
	})
}

func TestDecodeErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		data []byte
	}{
		{
			name: "corrupt header",
			data: []byte{0, 0, 2, 0, 0, 0},
		},
		{
			name: "no entries",
			data: []byte{0, 0, 1, 0, 0, 0},
		},
		{
			name: "short entry",
			data: []byte{0, 0, 1, 0, 1, 0},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if _, err := Decode(bytes.NewReader(tt.data)); err == nil {
				t.Fatal("expected error")
			}
			if _, err := DecodeAll(bytes.NewReader(tt.data)); err == nil {
				t.Fatal("expected DecodeAll error")
			}
		})
	}
}

func TestDecodeBMPHeadError(t *testing.T) {
	t.Parallel()

	dib := makeBMPDIB(maxSize+1, 1, 32, nil, nil)
	data := buildICO(dib, direntry{Width: 1, Height: 1, Plane: 1, Bits: 32, Size: uint32(len(dib)), Offset: 22})
	if _, err := Decode(bytes.NewReader(data)); err == nil {
		t.Fatal("expected forge BMP header error")
	}
}

func TestDecodeEntrySizeLimit(t *testing.T) {
	t.Parallel()

	data := buildICO(nil, direntry{
		Width:  1,
		Height: 1,
		Plane:  1,
		Bits:   32,
		Size:   maxEntrySize + 1,
		Offset: 22,
	})

	if _, err := Decode(bytes.NewReader(data)); err == nil {
		t.Fatal("expected Decode size limit error")
	}
	if _, err := DecodeAll(bytes.NewReader(data)); err == nil {
		t.Fatal("expected DecodeAll size limit error")
	}
	if _, err := DecodeConfig(bytes.NewReader(data)); err == nil {
		t.Fatal("expected DecodeConfig size limit error")
	}
}

func TestDecodeEntryCountLimit(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, head{Type: 1, Number: maxEntries + 1}); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()

	if _, err := Decode(bytes.NewReader(data)); err == nil {
		t.Fatal("expected Decode entry count limit error")
	}
	if _, err := DecodeAll(bytes.NewReader(data)); err == nil {
		t.Fatal("expected DecodeAll entry count limit error")
	}
	if _, err := DecodeConfig(bytes.NewReader(data)); err == nil {
		t.Fatal("expected DecodeConfig entry count limit error")
	}
}

func TestDecodeInvalidPNG(t *testing.T) {
	t.Parallel()

	invalidPNG := append(append([]byte{}, pngHeader...), 0)
	data := buildICO(invalidPNG, direntry{
		Width:  1,
		Height: 1,
		Plane:  1,
		Bits:   32,
		Size:   uint32(len(invalidPNG)),
		Offset: 22,
	})
	if _, err := Decode(bytes.NewReader(data)); err == nil {
		t.Fatal("expected png decode error")
	}
}

func TestForgeBMPHead(t *testing.T) {
	t.Parallel()

	t.Run("indexed color offset", func(t *testing.T) {
		t.Parallel()
		buf := makeBMPBuffer(12, 1, 1, 8, 44)
		mask, err := new(decoder).forgeBMPHead(buf, &direntry{Width: 1, Height: 1, Bits: 8})
		if err != nil {
			t.Fatal(err)
		}
		if len(mask) != 4 {
			t.Fatalf("expected 4-byte mask, got %d", len(mask))
		}
		if got := binary.LittleEndian.Uint32(buf[10:14]); got != 14+12+256*3 {
			t.Fatalf("unexpected offset: %d", got)
		}
	})

	t.Run("too large", func(t *testing.T) {
		t.Parallel()
		buf := makeBMPBuffer(40, maxSize+1, 1, 32, 40)
		if _, err := new(decoder).forgeBMPHead(buf, &direntry{Width: 1, Height: 1, Bits: 32}); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("mask larger than image", func(t *testing.T) {
		t.Parallel()
		buf := makeBMPBuffer(40, 10, 10, 1, 40)
		mask, err := new(decoder).forgeBMPHead(buf, &direntry{Width: 10, Height: 10, Bits: 1})
		if err != nil {
			t.Fatal(err)
		}
		if mask != nil {
			t.Fatalf("expected nil mask, got %d bytes", len(mask))
		}
	})

	t.Run("extended header profile data", func(t *testing.T) {
		t.Parallel()
		buf := makeBMPBuffer(108, 1, 1, 32, 108)
		binary.LittleEndian.PutUint32(buf[14+100:14+104], 16)
		if _, err := new(decoder).forgeBMPHead(buf, &direntry{Width: 1, Height: 1, Bits: 32}); err != nil {
			t.Fatal(err)
		}
		if got := binary.LittleEndian.Uint32(buf[10:14]); got != 14+108+16 {
			t.Fatalf("unexpected offset: %d", got)
		}
	})
}

func buildBMP32ICO() []byte {
	dib := makeBMPDIB(2, 4, 32, []byte{
		0, 0, 255, 255, 0, 255, 0, 128,
		255, 0, 0, 64, 255, 255, 255, 0,
	}, nil)
	return buildICO(dib, direntry{Width: 2, Height: 2, Plane: 1, Bits: 32, Size: uint32(len(dib)), Offset: 22})
}

func buildBMP24ICOWithMask() []byte {
	dib := makeBMPDIB(2, 4, 24, []byte{
		0, 0, 255, 0, 255, 0, 0, 0,
		255, 0, 0, 255, 255, 255, 0, 0,
	}, []byte{
		0x00, 0x00, 0x00, 0x00,
		0x40, 0x00, 0x00, 0x00,
	})
	return buildICO(dib, direntry{Width: 2, Height: 2, Plane: 1, Bits: 24, Size: uint32(len(dib)), Offset: 22})
}

func makeBMPDIB(width, height uint32, bits uint16, pixels, mask []byte) []byte {
	dib := make([]byte, 40+len(pixels)+len(mask))
	binary.LittleEndian.PutUint32(dib[0:4], 40)
	binary.LittleEndian.PutUint32(dib[4:8], width)
	binary.LittleEndian.PutUint32(dib[8:12], height)
	binary.LittleEndian.PutUint16(dib[12:14], 1)
	binary.LittleEndian.PutUint16(dib[14:16], bits)
	binary.LittleEndian.PutUint32(dib[20:24], uint32(len(pixels)))
	copy(dib[40:], pixels)
	copy(dib[40+len(pixels):], mask)
	return dib
}

func buildICO(data []byte, entry direntry) []byte {
	var buf bytes.Buffer
	if err := binary.Write(&buf, binary.LittleEndian, head{Type: 1, Number: 1}); err != nil {
		panic(err)
	}
	if err := binary.Write(&buf, binary.LittleEndian, entry); err != nil {
		panic(err)
	}
	buf.Write(data)
	return buf.Bytes()
}

func makeBMPBuffer(dibSize, width, height uint32, bits uint16, dataSize int) []byte {
	buf := make([]byte, 14+dataSize)
	data := buf[14:]
	binary.LittleEndian.PutUint32(data[0:4], dibSize)
	binary.LittleEndian.PutUint32(data[4:8], width)
	binary.LittleEndian.PutUint32(data[8:12], height)
	binary.LittleEndian.PutUint16(data[14:16], bits)
	return buf
}
