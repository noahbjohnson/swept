package main

import (
	"bufio"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"github.com/noahbjohnson/go-gpsd"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"xorm.io/xorm"
)

const sweepAlias = "hackrf_sweep"

// frequencyStringToInt parses a string into an integer representing hertz
func frequencyStringToInt(x string) (num int) {
	var err error
	num, err = strconv.Atoi(strings.Split(x, ".")[0])
	errPanic(err)
	return
}

// calculateBinRange calculates the highest and lowest frequencies in a bin
func calculateBinRange(hzLow int, hzHigh int, hzBinWidth int, binNum int) (low, high int) {
	low = hzLow + (binNum * hzBinWidth)
	high = low + hzBinWidth
	if high > hzHigh {
		high = hzHigh
	}
	return
}

// constructSweepArgs constructs arguments array for the scanRow call
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

// errPanic panics if passed an error otherwise just save me from repeating this damn code
// todo: eventually this should probably handle errors...
func errPanic(err error) {
	if err != nil {
		panic(err)
	}
}

// scanRow scans a row from the provided scanner and parses it
// breaks one row with multiple bin values into multiple rows with one bin each
// Append extracted rows to row array
func scanRow(scanner *bufio.Scanner, lat float64, lon float64, alt float64) (rows []Sample) {
	var row = strings.Split(scanner.Text(), ", ")
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
		insertRow := Sample{
			HzLow:          low,
			HzHigh:         high,
			Decibels:       decibels,
			N:              samples,
			Timestamp:      parsedTime,
			AltitudeMeters: alt,
			Latitude:       lat,
			Longitude:      lon,
		}
		rows = append(rows, insertRow)
	}
	return rows
}

type Sample struct {
	Id             int64 `xorm:"pk autoincr"`
	HzLow          int   `xorm:"index"`
	HzHigh         int   `xorm:"index"`
	Decibels       float64
	Latitude       float64 `xorm:"index"`
	Longitude      float64 `xorm:"index"`
	AltitudeMeters float64
	N              int
	Timestamp      time.Time `xorm:"index"`
}

// setupEngine creates a new orm engine and syncs the tables
func setupEngine() (engine *xorm.Engine) {
	engine, err := xorm.NewEngine("sqlite3", "./test.db")
	errPanic(err)
	err = engine.Sync2(new(Sample)) // Set up db tables
	errPanic(err)
	return
}

// setupCommand creates a hackrf_sweep command and the stdout scanner
func setupCommand() (cmd *exec.Cmd, scanner *bufio.Scanner) {
	cmd = exec.Command(sweepAlias, constructSweepArgs(false, 1000000)...)
	out, err := cmd.StdoutPipe()
	errPanic(err)
	scanner = bufio.NewScanner(out)
	return
}

// insertSampleRows inserts rows of samples in a transaction
func insertSampleRows(engine *xorm.Engine, rows []Sample) {
	sess := engine.NewSession()
	defer sess.Close()
	_, err := sess.Insert(rows)
	errPanic(err)
	err = sess.Commit()
	errPanic(err)
}

// todo: check that there is a hackrf plugged in
// todo: wait for gpsd tpv
// todo: use buffer channels like an adult
func main() {
	var (
		err      error
		rows     int
		start    = time.Now()
		laps     int // the lap number starting when updated
		lapLimit = 10
		lat      = new(float64)
		lon      = new(float64)
		alt      = new(float64)
	)

	// Set up command and db engine
	cmd, scanner := setupCommand()
	engine := setupEngine()

	gps, err := gpsd.Dial(gpsd.DefaultAddress)
	errPanic(err)
	gps.Subscribe("TPV", func(r interface{}) {
		tpv := r.(*gpsd.TPVReport)
		lat = &tpv.Lat
		lon = &tpv.Lon
		alt = &tpv.Alt
	})

	err = cmd.Start() // Start (async) command
	errPanic(err)

	gps.Run()
	defer gps.Close()

	for scanner.Scan() {
		newRows := scanRow(scanner, *lat, *lon, *alt)
		insertSampleRows(engine, newRows)

		if newRows[0].HzLow == 0 { // todo: update 0 to lower limit when implemented
			laps = laps + 1
			if laps > 1 {
				go logLaps(time.Since(start).Milliseconds(), rows, *lat, *lon)
			}
			if laps > lapLimit {
				break
			}
		}

		rows = rows + len(newRows)
	}
}

func logLaps(milliseconds int64, rows int, lat float64, lon float64) {
	fmt.Println(fmt.Sprintf("%d samples processed in %g seconds. current location: %f,%f", rows, float64(milliseconds)/1000, lat, lon))
}
