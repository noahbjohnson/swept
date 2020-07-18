package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const sweepAlias = "hackrf_sweep"

// Parses a string into an integer representing hertz
func frequencyStringToInt(x string) (num int) {
	var err error
	num, err = strconv.Atoi(strings.Split(x, ".")[0])
	errPanic(err)
	return
}

// Calculates the highest and lowest frequencies in a bin
func calculateBinRange(hzLow int, hzHigh int, hzBinWidth int, binNum int) (low, high int) {
	low = hzLow + (binNum * hzBinWidth)
	high = low + hzBinWidth
	if high > hzHigh {
		high = hzHigh
	}
	return
}

// construct arguments array for the sweep call
// todo: default bin size to 1000000 (1 million hertz)
// todo: high and low limits
// todo: sample rate
// one-shot mode (single sweep)
// bin width in hertz
func constructSweepArgs(oneShot bool, binSize int) (arguments []string) {
	if oneShot {
		arguments = append(arguments, "-1")
	}
	arguments = append(arguments, fmt.Sprintf("-w %v", binSize))
	return
}

// panic if passed an error otherwise just save me from repeating this damn code
// eventually this should probably handle errors...
func errPanic(err error) {
	if err != nil {
		panic(err)
	}
}

func sweep() ([][]string, time.Duration) {
	// Timer
	start := time.Now()
	var runtime time.Duration

	// Setup Command
	cmd := exec.Command(sweepAlias, constructSweepArgs(true, 1000000)...)
	out, err := cmd.StdoutPipe()
	errPanic(err)
	err = cmd.Start()
	errPanic(err)
	var rows [][]string

	// line parser for the stdout
	scanner := bufio.NewScanner(out)

	for scanner.Scan() {
		// Parse row
		rowString := scanner.Text()
		var row = strings.Split(rowString, ", ")
		var numBins = len(row) - 6
		var samples = frequencyStringToInt(row[5])
		// break row into bins
		for i := 0; i < numBins; i++ {
			var low, high = calculateBinRange(
				frequencyStringToInt(row[2]),
				frequencyStringToInt(row[3]),
				frequencyStringToInt(row[4]),
				i)
			var binRowIndex = i + 6
			parsedTime, err := time.Parse(time.RFC3339, row[0]+"T"+row[1]+"Z")
			errPanic(err)
			decibels := row[binRowIndex]
			insertRow := []string{
				strconv.Itoa(low),
				strconv.Itoa(high),
				decibels,
				strconv.Itoa(samples),
				parsedTime.String()}
			rows = append(rows, insertRow)
		}
	}
	runtime = time.Since(start)
	return rows, runtime
}

/*
todo: parse args from cli
*/
func main() {
	var laps []float64
	for i := 0; i < 101; i++ {
		_, duration := sweep()
		laps = append(laps, float64(duration.Milliseconds()))
		logLaps(laps)
	}
}

func logLaps(laps []float64) {
	if len(laps)%10 == 0 {
		var max = laps[0]
		var min = laps[0]
		var sum float64 = 0
		for i := 0; i < len(laps); i++ {
			if laps[i] > max {
				max = laps[i]
			} else if laps[i] < min {
				min = laps[i]
			}
			sum = sum + laps[i]
		}
		fmt.Printf("max: %g min: %g average: %g", max/1000, min/1000, (sum/float64(len(laps)))/1000)
		fmt.Println()
	}
}
