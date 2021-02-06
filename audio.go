package main

import (
	"bytes"
	"encoding/binary"
	"github.com/bobertlo/go-mpg123/mpg123"
	"github.com/gordonklaus/portaudio"
)

const (
	warningFileName  = "./warning.mp3"
	streamBufferSize = 8192
)

func warning() error {
	// create mpg123 decoder instance
	decoder, err := mpg123.NewDecoder("")
	if err != nil {
		return err
	}

	if err := decoder.Open(warningFileName); err != nil {
		return err
	}
	defer deferWithPrintError(decoder.Close)

	// get audio format information
	rate, channels, _ := decoder.GetFormat()

	// make sure output format does not change
	decoder.FormatNone()
	decoder.Format(rate, channels, mpg123.ENC_SIGNED_16)

	out := make([]int16, streamBufferSize)
	stream, err := portaudio.OpenDefaultStream(0, channels, float64(rate), len(out), &out)
	if err != nil {
		return err
	}
	defer deferWithPrintError(stream.Close)

	if err := stream.Start(); err != nil {
		return err
	}
	defer deferWithPrintError(stream.Stop)

	for {
		audio := make([]byte, 2*len(out))
		_, err = decoder.Read(audio)
		if err != nil {
			if err != mpg123.EOF {
				return err
			}
			break
		}
		if err := binary.Read(bytes.NewBuffer(audio), binary.LittleEndian, out); err != nil {
			return err
		}
		if err := stream.Write(); err != nil {
			return err
		}
	}
	return nil
}
