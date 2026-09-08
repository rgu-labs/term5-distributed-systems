package delivery

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"net"

	"github.com/rgu-labs/term5-distributed-systems/lab-1/internal/domain"
	"github.com/rgu-labs/term5-distributed-systems/lab-1/internal/service"
	"github.com/rgu-labs/term5-distributed-systems/lib/log"
)

const (
	sigmaSize   = 8
	headerSize  = 4
	maxFileSize = uint32(10 << 20)
)

var errInvalidFrame = errors.New("invalid frame")

type BlurHandler struct {
	svc *service.BlurService
}

func NewBlurHandler(svc *service.BlurService) *BlurHandler {
	return &BlurHandler{svc: svc}
}

func (h *BlurHandler) HandleConn(_ context.Context, conn net.Conn) {
	remote := conn.RemoteAddr().String()

	sigma, err := readFloat64(conn)
	if err != nil {
		log.Error("failed to read sigma", "remote", remote, "err", err)
		return
	}

	imgData, err := readBytes(conn)
	if err != nil {
		log.Error("failed to read image", "remote", remote, "err", err)
		return
	}

	rawImg := domain.RawImage{Data: imgData}
	img, err := domain.NewImageFromRaw(&rawImg)
	if err != nil {
		log.Error("failed to decode image", "err", err)
		return
	}

	blurred := h.svc.Blur(&img, sigma)

	encoded, err := blurred.Encode()
	if err != nil {
		log.Error("failed to encode image", "err", err)
		return
	}

	if err := writeBytes(conn, encoded.Data); err != nil {
		log.Error("failed to send result", "remote", remote, "err", err)
		return
	}
}

func readFloat64(r io.Reader) (float64, error) {
	var buf [sigmaSize]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("read sigma: %w", err)
	}
	return math.Float64frombits(binary.LittleEndian.Uint64(buf[:])), nil
}

func readBytes(r io.Reader) ([]byte, error) {
	var lenBuf [headerSize]byte
	if _, err := io.ReadFull(r, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	n := binary.LittleEndian.Uint32(lenBuf[:])
	if n > maxFileSize {
		return nil, fmt.Errorf("%w: length %d too large", errInvalidFrame, n)
	}

	buf := make([]byte, n)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	return buf, nil
}

func writeBytes(w io.Writer, data []byte) error {
	n := len(data)
	if n > int(maxFileSize) {
		return fmt.Errorf("%w: length %d too large", errInvalidFrame, n)
	}

	var lenBuf [headerSize]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(n))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("write length: %w", err)
	}
	if _, err := w.Write(data); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}
	return nil
}
