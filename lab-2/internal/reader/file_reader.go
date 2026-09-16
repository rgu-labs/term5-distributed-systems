package reader

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
)

type FileReader struct {
	mu      sync.Mutex
	scanner *bufio.Scanner
	file    io.ReadCloser
	err     error
}

func Open(path string) (*FileReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	sc := bufio.NewScanner(f)
	sc.Split(bufio.ScanWords)
	return &FileReader{
		file:    f,
		scanner: sc,
	}, nil
}

func (r *FileReader) Next() (int, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err != nil || !r.scanner.Scan() {
		if r.err == nil {
			r.err = r.scanner.Err()
		}
		return 0, false
	}

	value, err := strconv.Atoi(r.scanner.Text())
	if err != nil {
		r.err = fmt.Errorf("parse %q as integer: %w", r.scanner.Text(), err)
		return 0, false
	}
	return value, true
}

func (r *FileReader) Err() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.err
}

func (r *FileReader) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.file.Close()
}

func Count(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("open %s: %w", path, err)
	}

	n, scanErr := countTokens(f)
	closeErr := f.Close()
	if scanErr != nil {
		return 0, fmt.Errorf("read %s: %w", path, scanErr)
	}
	if closeErr != nil {
		return 0, fmt.Errorf("close %s: %w", path, closeErr)
	}
	return n, nil
}

func countTokens(r io.Reader) (int, error) {
	sc := bufio.NewScanner(r)
	sc.Split(bufio.ScanWords)

	n := 0
	for sc.Scan() {
		n++
	}
	return n, sc.Err()
}
