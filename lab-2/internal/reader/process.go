package reader

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"time"
)

type Strategy struct {
	ID    int
	Desc  string
	Apply func(n, a int) string
}

type StrategyStat struct {
	ID        int
	Processed int
}

type Report struct {
	Strategies    []StrategyStat
	TotalNumbers  int
	TotalOps      int
	Duration      time.Duration
}

func DefaultStrategies() []Strategy {
	return []Strategy{
		{
			ID:    1,
			Desc:  "next prime number",
			Apply: func(n, a int) string { return strconv.Itoa(Strategy1(n, a)) },
		},
		{
			ID:    2,
			Desc:  "checking divisibility by a",
			Apply: func(n, a int) string { return strconv.Itoa(Strategy2(n, a)) },
		},
		{
			ID:    3,
			Desc:  "next Fibonacci number",
			Apply: func(n, a int) string { return strconv.Itoa(Strategy3(n, a)) },
		},
		{
			ID:    4,
			Desc:  "Pythagorean triple check",
			Apply: func(n, a int) string { _, line := Strategy4(n, a); return line },
		},
	}
}

func Process(out io.Writer, r *FileReader, a int, strategies []Strategy) (*Report, error) {
	if r == nil {
		return nil, errors.New("reader is nil")
	}
	if len(strategies) == 0 {
		return nil, errors.New("at least one strategy is required")
	}

	start := time.Now()

	lines := make(chan string)
	printerDone := make(chan struct{})

	go func() {
		defer close(printerDone)
		for line := range lines {
			_, _ = fmt.Fprintln(out, line)
		}
	}()

	done := make(chan StrategyStat, len(strategies))
	for _, s := range strategies {
		go func(s Strategy) {
			processed := 0
			for {
				n, ok := r.Next()
				if !ok {
					break
				}
				lines <- fmt.Sprintf("[Strategy %d] number %d -> %s", s.ID, n, s.Apply(n, a))
				processed++
			}
			done <- StrategyStat{ID: s.ID, Processed: processed}
		}(s)
	}

	rep := &Report{Strategies: make([]StrategyStat, 0, len(strategies))}
	for range strategies {
		rep.Strategies = append(rep.Strategies, <-done)
	}
	rep.Duration = time.Since(start)

	sort.Slice(rep.Strategies, func(i, j int) bool {
		return rep.Strategies[i].ID < rep.Strategies[j].ID
	})
	for _, st := range rep.Strategies {
		rep.TotalOps += st.Processed
	}
	rep.TotalNumbers = rep.TotalOps

	close(lines)
	<-printerDone

	if err := r.Err(); err != nil {
		return nil, fmt.Errorf("read input: %w", err)
	}
	return rep, nil
}