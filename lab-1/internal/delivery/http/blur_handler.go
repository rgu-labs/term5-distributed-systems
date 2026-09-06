package delivery

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/rgu-labs/term5-distributed-systems/lab-1/internal/domain"
	"github.com/rgu-labs/term5-distributed-systems/lab-1/internal/service"
	"github.com/rgu-labs/term5-distributed-systems/lib/http/response"
)

const MaxBlurSigma = 20.0

type BlurHandler struct {
	svc *service.BlurService
}

func NewBlurHandler(svc *service.BlurService) *BlurHandler {
	return &BlurHandler{svc: svc}
}

func (h *BlurHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// NOTE(koftamainee): applyed middleware that cut off requests with more
	// than 10 << 20 bytes
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		response.BadRequest(w, "invalid multipart form")
		return
	}

	sigmaStr := r.FormValue("sigma")
	sigma, err := strconv.ParseFloat(sigmaStr, 64)
	if err != nil {
		response.BadRequest(w, "invalid sigma parameter")
		return
	}

	if sigma <= 0.0 {
		response.BadRequest(w, "sigma should be positive")
		return
	}

	if sigma > MaxBlurSigma {
		response.BadRequest(w,
			fmt.Sprintf("sigma should be less or equal to %.2f", MaxBlurSigma))
		return
	}

	file, header, err := r.FormFile("image")
	if err != nil {
		response.BadRequest(w, "image file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		response.Internal(w)
		return
	}

	rawImg := domain.RawImage{
		ContentType: header.Header.Get("Content-Type"),
		Data:        data,
	}

	img, err := domain.NewImageFromRaw(&rawImg)
	if err != nil {
		response.BadRequest(w, "failed to decode image: "+err.Error())
		return
	}

	blurred := h.svc.Blur(&img, sigma)

	encoded, err := blurred.Encode()
	if err != nil {
		response.Internal(w)
		return
	}

	w.Header().Set("Content-Type", encoded.ContentType)
	w.WriteHeader(http.StatusOK)
	w.Write(encoded.Data)
}
