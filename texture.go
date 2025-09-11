package mst

import (
	"bytes"
	"compress/zlib"
	"errors"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
)

// Texture 纹理结构体
type Texture struct {
	Id         int32     `json:"id"`
	Name       string    `json:"name"`
	Size       [2]uint64 `json:"size"`
	Format     uint16    `json:"format"`
	Type       uint16    `json:"type"`
	Compressed uint16    `json:"compressed"`
	Data       []byte    `json:"-"`
	Repeated   bool      `json:"repeated"`
}

func CompressImage(buf []byte) []byte {
	var bt []byte
	bf := bytes.NewBuffer(bt)
	w := zlib.NewWriter(bf)
	w.Write(buf)
	w.Close()
	return bf.Bytes()
}

func DecompressImage(src []byte) ([]byte, error) {
	bf := bytes.NewBuffer(src)
	r, er := zlib.NewReader(bf)
	if er != nil {
		return nil, er
	}
	return io.ReadAll(r)
}

func LoadTexture(tex *Texture, flipY bool) (image.Image, error) {
	w := int(tex.Size[0])
	h := int(tex.Size[1])
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	data := tex.Data
	var sz int
	switch tex.Format {
	case TEXTURE_FORMAT_RGB:
		sz = 3
	case TEXTURE_FORMAT_RGBA:
		sz = 4
	case TEXTURE_FORMAT_R:
		sz = 1
	}
	var e error
	if tex.Compressed == TEXTURE_COMPRESSED_ZLIB {
		data, e = DecompressImage(data)
		if e != nil && e.Error() != "EOF" {
			return nil, e
		}
	}

	for i := 0; i < h; i++ {
		for j := 0; j < w; j++ {
			p := i*w*sz + j*sz
			var c color.NRGBA
			switch sz {
			case 4:
				c = color.NRGBA{R: data[p], G: data[p+1], B: data[p+2], A: data[p+3]}
			case 3:
				c = color.NRGBA{R: data[p], G: data[p+1], B: data[p+2], A: 255}
			case 1:
				c = color.NRGBA{R: data[p], G: data[p], B: data[p], A: 255}
			}

			y := i
			if flipY {
				y = h - i - 1
			}
			img.Set(j, y, c)
		}
	}
	return img, nil
}

func CreateTexture(name string, repet bool) (*Texture, error) {
	reader, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	_, format, err := image.DecodeConfig(reader)
	if err != nil {
		return nil, err
	}
	reader.Seek(0, io.SeekStart)
	var img image.Image
	switch format {
	case "jpeg", "jpg":
		img, err = jpeg.Decode(reader)
	case "png":
		img, err = png.Decode(reader)
	case "gif":
		img, err = gif.Decode(reader)
	case "bmp":
		img, err = bmp.Decode(reader)
	case "tif", "tiff":
		img, err = tiff.Decode(reader)
	default:
		return nil, errors.New("unknow format")
	}
	if err != nil {
		return nil, err
	}
	return CreateTextureFromImage(img, name, repet)
}

func CreateTextureFromImage(img image.Image, name string, repet bool) (*Texture, error) {
	bd := img.Bounds()
	buf1 := []byte{}

	for y := 0; y < bd.Dy(); y++ {
		for x := 0; x < bd.Dx(); x++ {
			cl := img.At(x, y)
			r, g, b, a := color.RGBAModel.Convert(cl).RGBA()
			buf1 = append(buf1, byte(r&0xff), byte(g&0xff), byte(b&0xff), byte(a&0xff))
		}
	}
	t := &Texture{}
	_, fn := filepath.Split(name)
	t.Name = fn
	t.Format = TEXTURE_FORMAT_RGBA
	t.Size = [2]uint64{uint64(bd.Dx()), uint64(bd.Dy())}
	t.Compressed = TEXTURE_COMPRESSED_ZLIB
	t.Data = CompressImage(buf1)
	t.Repeated = repet
	return t, nil
}
