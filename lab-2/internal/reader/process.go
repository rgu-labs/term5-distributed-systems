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
	Strategies   []StrategyStat
	TotalNumbers int
	TotalOps     int
	Duration     time.Duration
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

func Process(out io.Writer, path string, a int, strategies []Strategy) (*Report, error) {
	if path == "" {
		return nil, errors.New("input file path is empty")
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

	type result struct {
		stat StrategyStat
		err  error
	}
	done := make(chan result, len(strategies))
	for _, s := range strategies {
		go func(s Strategy) {
			r, err := Open(path)
			if err != nil {
				done <- result{err: fmt.Errorf("open input file: %w", err)}
				return
			}
			defer r.Close()

			processed := 0
			for {
				n, ok := r.Next()
				if !ok {
					break
				}
				lines <- fmt.Sprintf("[Strategy %d] number %d -> %s", s.ID, n, s.Apply(n, a))
				processed++
			}
			if err := r.Err(); err != nil {
				done <- result{err: fmt.Errorf("read input: %w", err)}
				return
			}
			done <- result{stat: StrategyStat{ID: s.ID, Processed: processed}}
		}(s)
	}

	rep := &Report{Strategies: make([]StrategyStat, 0, len(strategies))}
	for range strategies {
		res := <-done
		if res.err != nil {
			return nil, res.err
		}
		rep.Strategies = append(rep.Strategies, res.stat)
	}
	rep.Duration = time.Since(start)

	sort.Slice(rep.Strategies, func(i, j int) bool {
		return rep.Strategies[i].ID < rep.Strategies[j].ID
	})
	for _, st := range rep.Strategies {
		rep.TotalOps += st.Processed
	}
	if len(rep.Strategies) > 0 {
		rep.TotalNumbers = rep.Strategies[0].Processed
	}

	close(lines)
	<-printerDone

	return rep, nil
}
