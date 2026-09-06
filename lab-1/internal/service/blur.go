package service

import (
	"image"
	"image/color"
	"math"

	"github.com/rgu-labs/term5-distributed-systems/lab-1/internal/domain"
)

type BlurService struct {
}

func NewBlur() *BlurService {
	return &BlurService{}
}

func (s *BlurService) Blur(img *domain.Image, sigma float64) domain.Image {
	if img == nil || img.Img == nil || sigma <= 0 {
		return domain.NewImage(colorImage(img.Img), img.Format())
	}

	kernel := gaussianKernel(sigma)
	blurred := blur2D(img.Img, kernel)
	return domain.NewImage(blurred, img.Format())
}

func gaussianKernel(sigma float64) []float64 {
	radius := int(math.Ceil(sigma*3.0)) + 1
	size := 2*radius + 1
	kernel := make([]float64, size)

	var sum float64
	for i := 0; i < size; i++ {
		x := float64(i - radius)
		kernel[i] = math.Exp(-(x * x) / (2 * sigma * sigma))
		sum += kernel[i]
	}
	for i := range kernel {
		kernel[i] /= sum
	}
	return kernel
}

func blur2D(src image.Image, kernel []float64) *image.NRGBA {
	bounds := src.Bounds()
	width, height := bounds.Dx(), bounds.Dy()

	horizontal := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			horizontal.SetNRGBA(x, y, convolveX(src, bounds, kernel, x, y))
		}
	}

	result := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			result.SetNRGBA(x, y, convolveY(horizontal, bounds, kernel, x, y))
		}
	}
	return result
}

func convolveX(src image.Image, bounds image.Rectangle, kernel []float64, x, y int) color.NRGBA {
	var r, g, b, a float64
	radius := len(kernel) / 2
	for k, w := range kernel {
		kx := clamp(x+k-radius, 0, bounds.Dx()-1)
		c := color.NRGBAModel.Convert(src.At(bounds.Min.X+kx, bounds.Min.Y+y)).(color.NRGBA)
		r += float64(c.R) * w
		g += float64(c.G) * w
		b += float64(c.B) * w
		a += float64(c.A) * w
	}
	return color.NRGBA{
		R: uint8(r + 0.5),
		G: uint8(g + 0.5),
		B: uint8(b + 0.5),
		A: uint8(a + 0.5),
	}
}

func convolveY(src *image.NRGBA, bounds image.Rectangle, kernel []float64, x, y int) color.NRGBA {
	var r, g, b, a float64
	radius := len(kernel) / 2
	for k, w := range kernel {
		ky := clamp(y+k-radius, 0, bounds.Dy()-1)
		c := src.NRGBAAt(x, ky)
		r += float64(c.R) * w
		g += float64(c.G) * w
		b += float64(c.B) * w
		a += float64(c.A) * w
	}
	return color.NRGBA{
		R: uint8(r + 0.5),
		G: uint8(g + 0.5),
		B: uint8(b + 0.5),
		A: uint8(a + 0.5),
	}
}

func colorImage(src image.Image) *image.NRGBA {
	if src == nil {
		return nil
	}
	bounds := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			out.Set(x, y, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return out
}

func clamp(v, min, max int) int {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
