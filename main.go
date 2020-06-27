package main

import (
	"bufio"
	"fmt"
	influxdb2 "github.com/influxdata/influxdb-client-go"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

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

func main() {
	var client = influxdb2.NewClientWithOptions("http://localhost:8086", "", influxdb2.DefaultOptions().SetLogLevel(3))
	var writeApi = client.WriteApi("", "rf")
	errorsCh := writeApi.Errors()
	go func() {
		for err := range errorsCh {
			fmt.Printf("write error: %s\n", err.Error())
		}
	}()

	// construct arguments array for the sweep call
	var arguments []string
	// one-shot mode (single sweep)
	//arguments = append(arguments, "-1")
	// enable rx amplifier
	arguments = append(arguments, "-a 1")
	// bin width in hertz
	//arguments = append(arguments, fmt.Sprintf("-w %v", 1000000))

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
	count := 0

	/*
		split row into multiple single-bin rows and insert them into influx
		[date, time, hz_low, hz_high, hz_bin_width, num_samples, bin1dB, bin2dB, bin3dB...]
		bin1 frequency range = (hz_low) > x < (hz_low + hz_bin_width)
		bin2 frequency range = (hz_low + hz_bin_width) > x < (hz_low + hz_bin_width * 2)
		etc...
	*/
	for scanner.Scan() {
		rowString := scanner.Text()
		var row = strings.Split(rowString, ", ")
		var numBins = len(row) - 6
		count = count + numBins
		var hzLow = frequencyStringToInt(row[2])
		var hzHigh = frequencyStringToInt(row[3])
		var hzBinWidth = frequencyStringToInt(row[4])
		var samples = frequencyStringToInt(row[5])
		for i := 0; i < numBins; i++ {
			var binRange = calculateBinRange(hzLow, hzHigh, hzBinWidth, i)
			var binRowIndex = i + 6
			var db float64
			var datetime time.Time
			var err error
			datetime, err = time.Parse(time.RFC3339, row[0]+"T"+row[1]+"Z")
			if err != nil {
				panic(err)
			}
			db, err = strconv.ParseFloat(row[binRowIndex], 64)
			if err != nil {
				panic(err)
			}
			p := influxdb2.NewPoint("rfdb",
				map[string]string{"hzLow": strconv.Itoa(binRange[0]),
					"hzHigh": strconv.Itoa(binRange[1])},
				map[string]interface{}{
					"samples": samples / numBins,
					"db":      db},
				datetime)
			writeApi.WritePoint(p)
			//fmt.Println(count)
		}
	}
	writeApi.Flush()
	client.Close()
}
