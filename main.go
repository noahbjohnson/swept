package main

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

var samples = 0

func sweep(widthHz int) {
	// construct arguments array for the sweep call
	var arguments []string
	// one-shot mode (single sweep)
	arguments = append(arguments, "-1")
	// enable rx amplifier
	arguments = append(arguments, "-a 1")
	// bin width in hertz
	arguments = append(arguments, fmt.Sprintf("-w %v", widthHz))

	// call the sweep with arguments
	cmd := exec.Command("hackrf_sweep", arguments...)

	// create new standard out pipe for the sweep
	out, err := cmd.StdoutPipe()
	if err != nil {
		panic(err)
	}

	// fire off sweep
	err = cmd.Start()
	if err != nil {
		panic(err)
	}

	// line parser for the stdout
	scanner := bufio.NewScanner(out)

	// split lines
	for scanner.Scan() {
		fmt.Println(splitRow(scanner.Text()))
	}
}

/*
Parses a string into an integer representing hertz
*/
func frequencyStringToInt(x string) int {
	var num, err = strconv.Atoi(strings.Split(x, ".")[0])
	if err != nil {
		panic(err)
	}
	return num
}

/*
Calculates the highest and lowest frequencies in a bin
*/
func calculateBinRange(hzLow int, hzHigh int, hzBinWidth int, binNum int) [2]int {
	var binOffset = binNum * hzBinWidth
	var low = hzLow + binOffset
	var high = low + hzBinWidth
	if high > hzHigh {
		high = hzHigh
	}
	return [2]int{low, high}
}

// split row into multiple single-bin rows
// [date, time, hz_low, hz_high, hz_bin_width, num_samples, bin1dB, bin2dB, bin3dB...]
// bin1 frequency range = (hz_low) > x < (hz_low + hz_bin_width)
// bin2 frequency range = (hz_low + hz_bin_width) > x < (hz_low + hz_bin_width * 2)
// etc...
func splitRow(rowString string) [][6]string {
	var row = strings.Split(rowString, ", ")
	// processed rows, one bin each
	// [date, time, hz_low, hz_high, samples (total samples / n-bins), db]
	var binRows [][6]string

	// 6 fields before the data values begin
	var numBins = len(row) - 6
	samples = samples + numBins
	var hzLow = frequencyStringToInt(row[2])
	var hzHigh = frequencyStringToInt(row[3])
	var hzBinWidth = frequencyStringToInt(row[4])
	var samples = frequencyStringToInt(row[5])

	for i := 0; i < numBins; i++ {
		var binRange = calculateBinRange(hzLow, hzHigh, hzBinWidth, i)
		var binRowIndex = i + 6
		var dataBinRow = [6]string{row[0], row[1], strconv.Itoa(binRange[0]), strconv.Itoa(binRange[1]), strconv.Itoa(samples / numBins), row[binRowIndex]}
		binRows = append(binRows, dataBinRow)
	}

	return binRows
}

func main() {
	// TODO:
	//   check hackrf with hackrf_info
	//   support rtl-sdr
	sweep(1000000)
}
