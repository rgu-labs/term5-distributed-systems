//nolint:forbidigo,gosec // CLI client prints plain TZ-format lines and dials a user-supplied address.
package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	transportHTTP = "http"
	transportTCP  = "tcp"

	defaultSigma float64 = 5.0

	sigmaSize    = 8
	headerSize   = 4
	maxImageSize = uint32(10 << 20)
)

func main() {
	sigma, positional, err := parseArgs(os.Args[1:])
	if err != nil {
		fatal(err.Error())
	}

	if len(positional) != 3 {
		fatal("usage: client <http|tcp> <host:port> <image> [-s|--sigma <float>]")
	}

	transport := positional[0]
	addr := positional[1]
	imagePath := positional[2]

	data, err := os.ReadFile(imagePath)
	if err != nil {
		fatal("failed to read image: " + err.Error())
	}

	switch transport {
	case transportTCP:
		if err := runTCP(addr, data, imagePath, sigma); err != nil {
			fatal(err.Error())
		}
	case transportHTTP:
		if err := runHTTP(addr, data, imagePath, sigma); err != nil {
			fatal(err.Error())
		}
	default:
		fatal("unknown transport: " + transport)
	}
}

func parseArgs(args []string) (float64, []string, error) {
	sigma := defaultSigma
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-s" || arg == "--sigma":
			if i+1 >= len(args) {
				return 0, nil, errors.New("option " + arg + " requires a value")
			}
			i++
			v, err := strconv.ParseFloat(args[i], 64)
			if err != nil {
				return 0, nil, fmt.Errorf("invalid sigma value %q", args[i])
			}
			sigma = v
		case strings.HasPrefix(arg, "--sigma="):
			v, err := strconv.ParseFloat(strings.TrimPrefix(arg, "--sigma="), 64)
			if err != nil {
				return 0, nil, fmt.Errorf("invalid sigma value %q", arg)
			}
			sigma = v
		case strings.HasPrefix(arg, "-s="):
			v, err := strconv.ParseFloat(strings.TrimPrefix(arg, "-s="), 64)
			if err != nil {
				return 0, nil, fmt.Errorf("invalid sigma value %q", arg)
			}
			sigma = v
		default:
			positional = append(positional, arg)
		}
	}

	return sigma, positional, nil
}

func runTCP(addr string, data []byte, imagePath string, sigma float64) error {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer conn.Close()

	fmt.Println("Connected to server")

	if err := writeFrame(conn, data, sigma); err != nil {
		return fmt.Errorf("failed to send image: %w", err)
	}
	fmt.Println("File sent: " + filepath.Base(imagePath))

	result, err := readFrame(conn)
	if err != nil {
		return fmt.Errorf("failed to receive result: %w", err)
	}
	fmt.Println("Result received")

	return save(result, imagePath)
}

func runHTTP(addr string, data []byte, imagePath string, sigma float64) error {
	var body bytes.Buffer

	mw := multipart.NewWriter(&body)
	if err := mw.WriteField("sigma", strconv.FormatFloat(sigma, 'f', -1, 64)); err != nil {
		return fmt.Errorf("failed to build form: %w", err)
	}
	fw, err := mw.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		return fmt.Errorf("failed to build form: %w", err)
	}
	if _, err := fw.Write(data); err != nil {
		return fmt.Errorf("failed to build form: %w", err)
	}
	if err := mw.Close(); err != nil {
		return fmt.Errorf("failed to build form: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "http://"+addr+"/api/blur", &body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer resp.Body.Close()

	fmt.Println("Connected to server")
	fmt.Println("File sent: " + filepath.Base(imagePath))

	if resp.StatusCode != http.StatusOK {
		b, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("server error: %d", resp.StatusCode)
		}
		return fmt.Errorf("server error: %d %s", resp.StatusCode, string(b))
	}

	result, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to receive result: %w", err)
	}
	fmt.Println("Result received")

	return save(result, imagePath)
}

func save(data []byte, imagePath string) error {
	out := "blurred_" + filepath.Base(imagePath)
	if err := os.WriteFile(out, data, 0o600); err != nil {
		return fmt.Errorf("failed to save result: %w", err)
	}
	fmt.Println("Saved: " + out)
	return nil
}

func writeFrame(conn net.Conn, data []byte, sigma float64) error {
	var sigmaBuf [sigmaSize]byte
	binary.LittleEndian.PutUint64(sigmaBuf[:], math.Float64bits(sigma))
	if _, err := conn.Write(sigmaBuf[:]); err != nil {
		return fmt.Errorf("write sigma: %w", err)
	}

	n := len(data)
	if n > int(maxImageSize) {
		return fmt.Errorf("image too large: %d bytes", n)
	}

	var lenBuf [headerSize]byte
	binary.LittleEndian.PutUint32(lenBuf[:], uint32(n))
	if _, err := conn.Write(lenBuf[:]); err != nil {
		return fmt.Errorf("write length: %w", err)
	}

	if _, err := conn.Write(data); err != nil {
		return fmt.Errorf("write image: %w", err)
	}
	return nil
}

func readFrame(conn net.Conn) ([]byte, error) {
	var lenBuf [headerSize]byte
	if _, err := io.ReadFull(conn, lenBuf[:]); err != nil {
		return nil, fmt.Errorf("read length: %w", err)
	}

	n := binary.LittleEndian.Uint32(lenBuf[:])
	if n > maxImageSize {
		return nil, errors.New("result too large")
	}

	buf := make([]byte, n)
	if _, err := io.ReadFull(conn, buf); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}
	return buf, nil
}

func fatal(msg string) {
	fmt.Println(msg)
	os.Exit(1)
}
