package domain

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
)

type Image struct {
	Img    image.Image
	format string // "jpeg", "png", "gif"
}

func NewImage(img image.Image, format string) Image {
	return Image{Img: img, format: format}
}

type RawImage struct {
	ContentType string
	Data        []byte
}

func NewImageFromRaw(rawImg *RawImage) (Image, error) {
	img, format, err := image.Decode(bytes.NewReader(rawImg.Data))
	if err != nil {
		return Image{}, fmt.Errorf("failed to decode image: %w", err)
	}
	return Image{Img: img, format: format}, nil
}

func (i *Image) Format() string {
	return i.format
}

func (i *Image) Encode() (RawImage, error) {
	var buf bytes.Buffer
	var err error
	var contentType string

	switch i.format {
	case "jpeg", "jpg":
		contentType = "image/jpeg"
		err = jpeg.Encode(&buf, i.Img, nil)
	case "png":
		contentType = "image/png"
		err = png.Encode(&buf, i.Img)
	case "gif":
		contentType = "image/gif"
		err = gif.Encode(&buf, i.Img, nil)
	default:
		err = fmt.Errorf("unsupported image format: %s", i.format)
	}
	if err != nil {
		return RawImage{}, fmt.Errorf("failed to encode image: %w", err)
	}
	return RawImage{ContentType: contentType, Data: buf.Bytes()}, nil
}
