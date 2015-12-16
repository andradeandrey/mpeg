package ts

import "io"
import "bytes"
import "github.com/32bitkid/mpeg/util"

type PayloadReader interface {
	io.Reader
	TransportStreamControl
}

func NewPayloadReader(source io.Reader, where PacketTester) PayloadReader {
	return &payloadReader{
		currentPacket: new(Packet),
		br:            util.NewBitReader(source),
		where:         where,
		closed:        false,
		isAligned:     false,
		skipUntil:     alwaysTrueTester,
		takeWhile:     alwaysTrueTester,
	}
}

type payloadReader struct {
	currentPacket *Packet
	br            util.BitReader32
	where         PacketTester
	skipUntil     PacketTester
	takeWhile     PacketTester
	remainder     bytes.Buffer
	closed        bool
	isAligned     bool
}

func (r *payloadReader) SkipUntil(skipUntil PacketTester) {
	r.skipUntil = r.where.And(skipUntil)
}

func (r *payloadReader) TakeWhile(takeWhile PacketTester) {
	r.takeWhile = r.where.And(takeWhile)
}

func (r *payloadReader) Read(p []byte) (n int, err error) {

	if r.closed == true {
		return 0, io.EOF
	}

	if r.isAligned == false {
		err = r.realign()
		if err != nil {
			return
		}
	}

	var remainder []byte

	// Drain remainder
	for len(p) > 0 {
		cn, err := r.remainder.Read(p)
		n = n + cn
		p = p[cn:]
		if err == io.EOF {
			break
		} else if err != nil {
			return n, err
		}
	}

	// Fill from packet stream
	for len(p) > 0 {
		err = r.next()
		if err != nil {
			return
		}

		if r.where(r.currentPacket) {
			copied := copy(p, r.currentPacket.Payload)
			n = n + copied
			p = p[copied:]
			remainder = r.currentPacket.Payload[copied:]
		}

		cont := r.takeWhile(r.currentPacket)
		if cont == false {
			r.closed = true
			return n, io.EOF
		}
	}

	_, err = r.remainder.Write(remainder)
	return
}

func (r *payloadReader) next() error {
	return r.currentPacket.ReadFrom(r.br)
}

func (r *payloadReader) realign() (err error) {
	for {
		r.next()
		if err != nil {
			return err
		}
		done := r.skipUntil(r.currentPacket)
		if done {
			r.isAligned = true
			r.remainder.Write(r.currentPacket.Payload)
			return nil
		}
	}
}
