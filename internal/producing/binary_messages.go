package producing

import (
	"encoding/binary"
	"hash/crc32"

	"github.com/polarstreams/polar/internal/conf"
	. "github.com/polarstreams/polar/internal/types"
	"github.com/polarstreams/polar/internal/utils"
)

type opcode uint8
type streamId uint16
type flags uint8

const messageVersion = 1

// Operation codes.
// Use fixed numbers (not iota) to make it harder to break the protocol by moving stuff around.
const (
	startupOp         opcode = 1
	readyOp           opcode = 2
	errorOp           opcode = 3
	produceOp         opcode = 4
	produceResponseOp opcode = 5
)

// Flags.
// Use fixed numbers (not iota) to make it harder to break the protocol by moving stuff around.
const (
	withTimestamp flags = 0b00000001
)

// Header for producer messages. Order of fields defines the serialization format.
type binaryHeader struct {
	Version    uint8
	Flags      flags
	StreamId   streamId
	Op         opcode
	BodyLength uint32
	Crc        uint32
}

var binaryHeaderSize = utils.BinarySize(binaryHeader{})

type binaryResponse interface {
	Marshal(w BufferBackedWriter) error
}

type emptyResponse struct {
	streamId streamId
	op       opcode
}

func (r *emptyResponse) Marshal(w BufferBackedWriter) error {
	return writeHeader(w, &binaryHeader{
		Version:    messageVersion,
		StreamId:   r.streamId,
		Op:         r.op,
		Flags:      0,
		BodyLength: 0,
		Crc:        0,
	})
}

type errorResponse struct {
	streamId streamId
	op       opcode
	message  string
}

func (r *errorResponse) Marshal(w BufferBackedWriter) error {
	message := []byte(r.message)
	if err := writeHeader(w, &binaryHeader{
		Version:    messageVersion,
		StreamId:   r.streamId,
		Op:         errorOp,
		BodyLength: uint32(len(message)),
	}); err != nil {
		return err
	}

	_, err := w.Write(message)
	return err
}

func writeHeader(w BufferBackedWriter, header *binaryHeader) error {
	if err := binary.Write(w, conf.Endianness, header); err != nil {
		return err
	}

	const crcByteSize = 4
	buf := w.Bytes()
	headerBuf := buf[len(buf)-binaryHeaderSize:]
	crc := crc32.ChecksumIEEE(headerBuf[:len(headerBuf)-crcByteSize])
	conf.Endianness.PutUint32(headerBuf[len(headerBuf)-crcByteSize:], crc)
	return nil
}

func newErrorResponse(message string, requestHeader *binaryHeader) binaryResponse {
	return &errorResponse{message: message, streamId: requestHeader.StreamId}
}
