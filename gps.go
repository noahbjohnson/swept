package main

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.bug.st/serial"
	"sort"
	"strings"
)

func GetPortString() (string, error) {
	ports, err := serial.GetPortsList()
	if err != nil {
		return "", err
	}
	if len(ports) == 0 {
		return "", fmt.Errorf("no serial ports found")
	}
	var portNames []string
	for _, port := range ports {
		if strings.Contains(port, "usbserial") && strings.Contains(port, "tty") {
			portNames = append(portNames, port)
		}
	}
	if len(portNames) == 0 {
		return "", fmt.Errorf("no serial ports found")
	}
	sort.Strings(portNames)
	portName := portNames[0]
	return portName, err
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal().Err(err).Send()
	}
	if len(ports) == 0 {
		log.Fatal().Msg("No serial ports found!")
	}
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
	}
	portName, err := GetPortString()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get serial port")
	}
	log.Info().Str("port", portName).Msg("usb device found")
	mode := &serial.Mode{
		BaudRate: 4800,
	}
	reader, err := serial.Open(portName, mode)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open serial port")
	}
	log.Info().Str("port", portName).Msg("Opened serial connection")
	buff := make(chan string, 1000)
	go func() {
		readerBuff := make([]byte, 100)
		for {
			n, err := reader.Read(readerBuff)
			if err != nil {
				log.Fatal().Err(err).Msg("buffered read failed")
				break
			}
			if n == 0 {
				fmt.Println("\nEOF")
				break
			}
			buff <- string(readerBuff[:n])
		}
	}()
	for v := range buff {
		fmt.Print(v)
	}
}
