package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/rgu-labs/term5-distributed-systems/lab-2/internal/reader"
)

const (
	divider     = "---------------------------------------"
	dividerBold = "======================================="
)

type config struct {
	input string
	param int
}

func main() {
	cfg := parseFlags()

	total, err := reader.Count(cfg.input)
	if err != nil {
		fatal(err.Error())
	}

	fr, err := reader.Open(cfg.input)
	if err != nil {
		fatal(err.Error())
	}

	strategies := reader.DefaultStrategies()

	reportHeader(cfg, total, len(strategies))
	reportInit(cfg.input, strategies)

	fmt.Println(divider)
	fmt.Println("START PROCESSING")
	fmt.Println(divider)

	rep, err := reader.Process(os.Stdout, fr, cfg.param, strategies)
	if err != nil {
		fatal("processing failed: " + err.Error())
	}

	reportStats(rep)
	reportPerformance(rep.Duration)

	if err := fr.Close(); err != nil {
		fatal("close input file: " + err.Error())
	}
}

func parseFlags() config {
	input := flag.String("file", "numbers.txt", "path to the integer input file")
	param := flag.Int("a", 5, "strategy parameter")
	flag.Parse()

	if *param <= 0 {
		fatal("the strategy parameter a must be a positive integer")
	}
	return config{input: *input, param: *param}
}

func reportHeader(cfg config, total, strategies int) {
	fmt.Println(dividerBold)
	fmt.Println("PARALLEL FILE PROCESSING")
	fmt.Println(dividerBold)
	fmt.Println("Input file:")
	fmt.Println(cfg.input)
	fmt.Println("Total numbers:")
	fmt.Println(total)
	fmt.Println("Number of strategies:")
	fmt.Println(strategies)
	fmt.Println("Number of goroutines:")
	fmt.Println(strategies)
	fmt.Println("Parameter a:")
	fmt.Println(cfg.param)
}

func reportInit(file string, strategies []reader.Strategy) {
	fmt.Println(divider)
	fmt.Println("INITIALIZATION")
	fmt.Println(divider)
	fmt.Println("FileReader initialized successfully.")
	fmt.Println("File: " + file)
	fmt.Println("Strategies:")
	for i, s := range strategies {
		fmt.Printf("Goroutine %d:\n", i+1)
		fmt.Printf("Strategy %d - %s\n", s.ID, s.Desc)
	}
}

func reportStats(rep *reader.Report) {
	fmt.Println(divider)
	fmt.Println("STATISTICS")
	fmt.Println(divider)
	for _, st := range rep.Strategies {
		fmt.Printf("Strategy %d processed: %d numbers\n", st.ID, st.Processed)
	}
	fmt.Printf("Total numbers processed: %d\n", rep.TotalNumbers)
	fmt.Printf("Total operations: %d\n", rep.TotalOps)
}

func reportPerformance(dur time.Duration) {
	fmt.Println(divider)
	fmt.Println("PERFORMANCE")
	fmt.Println(divider)
	fmt.Println("Execution time:")
	fmt.Printf("%.3f ms\n", float64(dur)/float64(time.Millisecond))
	fmt.Println("All goroutines completed successfully.")
	fmt.Println(dividerBold)
	fmt.Println("PROGRAM FINISHED")
	fmt.Println(dividerBold)
}

func fatal(message string) {
	fmt.Fprintln(os.Stderr, message)
	os.Exit(1)
}
