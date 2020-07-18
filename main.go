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

// construct arguments array for the scanRow call
// todo: default bin size to 1000000 (1 million hertz)
// todo: high and low limits
// todo: sample rate
// one-shot mode (single scanRow)
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

func scanRow(scanner *bufio.Scanner) (rows [][]string, runtime time.Duration) {
	// Timer
	start := time.Now()

	scanner.Scan()
	rowString := scanner.Text()
	rows = parseRow(rows, rowString)
	runtime = time.Since(start)
	return
}

// Break one row with multiple bin values into multiple rows with one bin each
// Append extracted rows to row array
func parseRow(rows [][]string, rowString string) [][]string {
	var row = strings.Split(rowString, ", ")
	var numBins = len(row) - 6
	var samples = frequencyStringToInt(row[5])
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
	return rows
}

func main() {
	var laps []float64
	// Setup Command
	cmd := exec.Command(sweepAlias, constructSweepArgs(true, 1000000)...)
	out, err := cmd.StdoutPipe()
	errPanic(err)
	err = cmd.Start()
	errPanic(err)

	scanner := bufio.NewScanner(out)
	var rows [][]string
	for i := 0; i < 10000; i++ {
		newRows, duration := scanRow(scanner)
		laps = append(laps, float64(duration.Milliseconds()))
		rows = append(rows, newRows...)
		logLaps(laps)
	}
	println(len(rows))
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
