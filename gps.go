package main

import (
	"fmt"
	"github.com/adrianmo/go-nmea"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.bug.st/serial"
	"sort"
	"strings"
)

type Command uint8

type CommandChannel chan Command
type StringChannel chan string

const (
	Kill Command = iota
)

type GpsController struct {
	port             serial.Port
	portName         string
	stringChannel    StringChannel
	lineChannel      StringChannel
	cleanLineChannel StringChannel
	controlChannel   CommandChannel
	// todo reader status?
}

func (s *GpsController) GetPortString() error {
	ports, err := serial.GetPortsList()
	if err != nil {
		return err
	}
	if len(ports) == 0 {
		return fmt.Errorf("no serial ports found")
	}
	var portNames []string
	for _, port := range ports {
		if strings.Contains(port, "usbserial") && strings.Contains(port, "tty") {
			portNames = append(portNames, port)
		}
	}
	if len(portNames) == 0 {
		return fmt.Errorf("no serial ports found")
	}
	sort.Strings(portNames)
	s.portName = portNames[0]
	log.Info().Str("port", s.portName).Msg("Port found")
	return err
}

func (s *GpsController) OpenPort() error {
	if s.portName == "" {
		return fmt.Errorf("no port was identified. run GetPortString first")
	}
	mode := &serial.Mode{
		BaudRate: 4800,
	}
	reader, err := serial.Open(s.portName, mode)
	if err != nil {
		return fmt.Errorf("failed to open serial port")
	}
	s.port = reader
	log.Info().Str("port", s.portName).Msg("Port opened")
	return nil
}

func (s *GpsController) createChannels() {
	if s.stringChannel == nil {
		s.stringChannel = make(StringChannel, 1024)
	}
	if s.lineChannel == nil {
		s.lineChannel = make(StringChannel, 1024)
	}
	if s.cleanLineChannel == nil {
		s.cleanLineChannel = make(StringChannel, 1024)
	}
	if s.controlChannel == nil {
		s.controlChannel = make(CommandChannel, 10)
	}
}

func (s *GpsController) Read() {
	s.createChannels()

	// read the serial bytes into a string channel
	s.read()

	// break up the strings into lines in the line channel
	s.parseLines()

	// clean lines of trailing spaces and invalid characters
	s.cleanLines()

	for v := range s.cleanLineChannel {
		sentence, err := nmea.Parse(v)
		if err != nil {
			log.Error().Err(err).Str("line", v).Send()
		}
		log.Debug().Msg(sentence.String())
	}

}

func (s *GpsController) read() {
	go func() {
		readerBuff := make([]byte, 2048)
		for {
			// todo check control channel
			n, err := s.port.Read(readerBuff)
			if err != nil {
				log.Fatal().Err(err).Msg("serial read failed")
				break
			}
			if n == 0 {
				log.Info().Msg("serial read ended")
				break
			}
			s.stringChannel <- string(readerBuff[:n])
		}
	}()
}

func (s *GpsController) parseLines() {
	go func() {
		stringBuff := make([]string, 1024)
		firstLine := true

		for v := range s.stringChannel {
			if strings.Contains(v, "\n") {
				split := strings.Split(v, "\n")
				stringBuff = append(stringBuff, split[0])
				if !firstLine {
					s.lineChannel <- strings.Join(stringBuff, "")
				} else {
					firstLine = false
				}
				for _, i := range split[1 : len(split)-1] {
					s.lineChannel <- i
				}
				stringBuff = make([]string, 1024)
				stringBuff = append(stringBuff, split[len(split)-1])
			}
			stringBuff = append(stringBuff, v)
		}

	}()
}

func (s GpsController) cleanLines() {
	go func() {
		for line := range s.lineChannel {
			s.cleanLineChannel <- strings.TrimSpace(line)
		}
	}()
}

func (s GpsController) Stop() {
	// todo
}

func main() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	controller := GpsController{}

	err := controller.GetPortString()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	err = controller.OpenPort()
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	controller.Read()
}
