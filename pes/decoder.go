package pes

import "log"
import "bytes"
import "github.com/32bitkid/mpeg-go/ts"
import "github.com/32bitkid/bitreader"

type PacketChannel <-chan *Packet

func (input PacketChannel) PayloadOnly() <-chan []byte {
	output := make(chan []byte)
	go func() {
		for packet := range input {
			output <- packet.Payload
		}
		close(output)
	}()
	return output
}

func TsDecoder(input ts.PacketChannel) PacketChannel {
	output := make(chan *Packet)

	buffer := &bytes.Buffer{}
	reader := bitreader.NewReader32(buffer)

	go func() {
		for tsPacket := range input {

			if tsPacket.PayloadUnitStartIndicator && buffer.Len() > 0 {

				// Drain
				pesPacket, err := ReadPacket(reader, buffer.Len())
				if err != nil {
					log.Println(err)
					close(output)
					return
				}
				output <- pesPacket
			}

			// Fill
			buffer.Write(tsPacket.Payload)
		}

		close(output)
	}()

	return output
}

func PayloadDecoder(input <-chan []byte) PacketChannel {
	output := make(chan *Packet)
	reader := bitreader.NewBufferedBitreader()
	closed := false

	// Fill
	go func() {
		for payload := range input {
			reader.Write(payload)
		}
		closed = true
	}()

	// Drain
	go func() {
		for !closed {
			packet, err := ReadPacket(reader, 0)
			if err != nil {
				log.Println(err)
				close(output)
				return
			}
			log.Printf("%b", packet.StreamID)
		}
	}()

	return output
}
