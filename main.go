package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const sweepAlias = "hackrf_sweep"

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

/*
construct arguments array for the sweep call
todo: default bin size to 1000000 (1 million hertz)
*/
func constructSweepArgs(oneShot bool, binSize int) []string {
	var arguments []string
	//if amplifier {
	// enable rx amplifier
	// LEAVE AMP OFF so you don't blow your board near a cell tower
	// only engage it if you're in the boonies
	//arguments = append(arguments, "-a 1")
	//}
	if oneShot {
		// one-shot mode (single sweep)
		arguments = append(arguments, "-1")
	}
	// bin width in hertz
	arguments = append(arguments, fmt.Sprintf("-w %v", binSize))
	return arguments
}

/*
panic if passed an error otherwise just save me from repeating this damn code
eventually this should probably handle errors...
*/
func errPanic(err error) {
	if err != nil {
		panic(err)
	}
}

/*
todo: parse args from cli
*/
func main() {
	// call the sweep with arguments
	// create new standard out pipe for the sweep
	// fire off sweep
	cmd := exec.Command(sweepAlias, constructSweepArgs(false, 1000000)...)
	out, err := cmd.StdoutPipe()
	errPanic(err)
	err = cmd.Start()
	errPanic(err)

	// line parser for the stdout
	scanner := bufio.NewScanner(out)
	count := 0

	outFile, err := os.Create("out.txt")
	errPanic(err)
	defer outFile.Close()

	var lastLap time.Time
	firstLap := true
	var laps []float64

	/*
		split row into multiple single-bin rows
		[date, time, hz_low, hz_high, hz_bin_width, num_samples, bin1dB, bin2dB, bin3dB...]
		bin1 frequency range = (hz_low) > x < (hz_low + hz_bin_width)
		bin2 frequency range = (hz_low + hz_bin_width) > x < (hz_low + hz_bin_width * 2)
		etc...
	*/
	for scanner.Scan() {
		// Parse row
		rowString := scanner.Text()
		var row = strings.Split(rowString, ", ")
		var numBins = len(row) - 6
		var hzLow = frequencyStringToInt(row[2])
		if hzLow == 0 {
			if firstLap {
				firstLap = false
				lastLap = time.Now()
			} else {
				laps = append(laps, float64(time.Now().Sub(lastLap).Milliseconds()))
				lastLap = time.Now()
				go func() {
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
				}()
			}
		}
		var hzHigh = frequencyStringToInt(row[3])
		var hzBinWidth = frequencyStringToInt(row[4])
		var samples = frequencyStringToInt(row[5])
		count = count + numBins
		// break row into bins
		for i := 0; i < numBins; i++ {
			var binRange = calculateBinRange(hzLow, hzHigh, hzBinWidth, i)
			var binRowIndex = i + 6
			var decibels float64
			var datetime time.Time
			var err error
			datetime, err = time.Parse(time.RFC3339, row[0]+"T"+row[1]+"Z")
			errPanic(err)
			decibels, err = strconv.ParseFloat(row[binRowIndex], 64)
			errPanic(err)
			// todo: write row here
			insertRow := []string{strconv.Itoa(binRange[0]), strconv.Itoa(binRange[1]), strconv.FormatFloat(decibels, 'f', -1, 64), strconv.Itoa(samples), datetime.String()}
			_, _ = outFile.Write([]byte(strings.Join(insertRow, ",")))
			outFile.Write([]byte("\n"))
		}
	}

}
