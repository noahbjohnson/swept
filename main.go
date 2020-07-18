package main

import (
	"bufio"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
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

func scanRow(scanner *bufio.Scanner) (rows []Row, runtime time.Duration) {
	// Timer
	start := time.Now()
	defer func() { runtime = time.Since(start) }()
	var rowString string
	scanner.Scan()
	rowString = scanner.Text()
	if len(rowString) < 1 {
		return
	}
	rows = parseRow(rows, rowString)
	return
}

type Row struct {
	Id        int64 `xorm:"pk autoincr"`
	HzLow     int   `xorm:"index"`
	HzHigh    int   `xorm:"index"`
	Decibels  float64
	N         int
	Timestamp time.Time
}

// Break one row with multiple bin values into multiple rows with one bin each
// Append extracted rows to row array
func parseRow(rows []Row, rowString string) []Row {
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
		decibels, err := strconv.ParseFloat(row[binRowIndex], 64)
		errPanic(err)
		insertRow := Row{
			HzLow:     low,
			HzHigh:    high,
			Decibels:  decibels,
			N:         samples,
			Timestamp: parsedTime,
		}
		rows = append(rows, insertRow)
	}
	return rows
}

func main() {
	var (
		laps float64
		rows []Row
		err  error
	)

	// Setup Command
	cmd := exec.Command(sweepAlias, constructSweepArgs(false, 1000000)...)
	out, err := cmd.StdoutPipe()
	errPanic(err)

	defer func() {
		_ = out.Close()
	}()

	//engine, err := xorm.NewEngine("sqlite3", "./test.db")
	//errPanic(err)

	scanner := bufio.NewScanner(out)
	err = cmd.Start()
	errPanic(err)

	for i := 0; i < 100000000; i++ {
		newRows, duration := scanRow(scanner)
		if len(newRows) < 1 {
			break
		}
		laps = laps + float64(duration.Milliseconds())
		rows = append(rows, newRows...)
		if len(rows)%10000 == 0 {
			go logLaps(laps, len(rows))
		}
	}
	logLaps(laps, len(rows))
	println(len(rows))
}

func logLaps(milliseconds float64, rows int) {
	fmt.Printf("%d samples processed in %g seconds", rows, milliseconds/1000)
	fmt.Println()
}
