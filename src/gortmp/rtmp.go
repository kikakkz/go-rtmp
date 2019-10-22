package rtmp

/*
#cgo LDFLAGS: -lrtmp
#include "librtmp/amf.h"
*/
import "C"

import (
	bin "encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	MaxHeaderSize    = 18
	DefaultChunkSize = 128
	Version          = 3
)

const (
	MsgChunkSize    = 1
	MsgStreamCancel = 2
	MsgSendWndSize  = 3
	MsgRecvWndSize  = 5
	MsgBandwidth    = 6
	MsgCmdAMF0      = 20
	MsgCmdAMF3      = 17
	MsgInfoAMF0     = 18
	MsgInfoAMF3     = 15
	MsgShareObjAMF0 = 19
	MsgShareObjAMF3 = 16
	MsgAudio        = 8
	MsgVideo        = 9
	MsgAggregated   = 22
)

/* Only for RTMP server */
const (
	HANDSHAKE_C0 = 0
	HANDSHAKE_C1 = 1
	HANDSHAKE_C2 = 2
	STREAMING    = 3
)

const (
	EV_READ_FAIL        = 0
	EV_FATAL            = 1
	EV_C0               = 2
	EV_MISMATCH_VERSION = 3
	EV_C1               = 4
	EV_ZERO_ERROR       = 5
	EV_C2               = 6
	EV_MISMATCH_TS      = 7
	EV_NEW_STREAM       = 8
	EV_DEL_STREAM       = 9
)

type packet struct {
	absTimestamp    bool
	timestamp       uint32
	extTimestamp    bool
	extTimestampVal uint32
	msgLen          int
	msgRead         int
	msgTypeID       uint8
	msgStreamID     uint32
	body            []byte
}

type stream struct {
	streamID int
	pkt      packet
}

type RTMP struct {
	stat         int
	conn         *net.TCPConn
	evl          func(ev int, body []byte) error
	tsC1         uint32
	tsS1         uint32
	tsReadC1     uint32
	streams      []*stream
	inChunkSize  int
	outChunkSize int
}

func New(conn *net.TCPConn, evl func(ev int, body []byte) error) *RTMP {
	var r RTMP

	r.conn = conn
	r.stat = HANDSHAKE_C0
	r.evl = evl
	r.streams = make([]*stream, 0)
	r.tsC1 = 0
	r.tsReadC1 = 0
	r.tsS1 = 0
	r.inChunkSize = DefaultChunkSize
	r.outChunkSize = DefaultChunkSize

	return &r
}

func (r *RTMP) write(b []byte) (int, error) {
	return r.conn.Write(b)
}

func (r *RTMP) read(b []byte) (int, error) {
	r.conn.SetReadDeadline(time.Now().Add(10 * time.Millisecond))
	n, err := r.conn.Read(b)
	if nil != err {
		if !r.netTimeout(err) {
			fmt.Printf("Fail read C1(%s)\n", err)
			r.event(EV_READ_FAIL, nil)
			return 0, err
		}
	}
	return n, nil
}

func (r *RTMP) event(ev int, b []byte) {
	if nil != r.evl {
		r.evl(ev, b)
	}
}

func (r *RTMP) netTimeout(err error) bool {
	netErr, ok := err.(net.Error)
	if ok && netErr.Timeout() {
		return true
	}
	return false
}

func (r *RTMP) onC0() error {
	var b [4 + 4 + 1528]byte
	_, err := r.read(b[0:1])
	if nil != err {
		return err
	}
	if Version != b[0] {
		str := fmt.Sprintf("Mismatch version(%d != %d)\n", Version, b[0])
		fmt.Printf(str)
		r.event(EV_MISMATCH_VERSION, nil)
		return errors.New(str)
	}

	r.write(b[0:1])

	timenow := uint32(time.Now().Unix())
	bin.BigEndian.PutUint32(b[0:4], timenow)
	r.tsS1 = timenow

	r.write(b[0:])
	r.event(EV_C0, nil)
	r.stat = HANDSHAKE_C1

	return nil
}

func (r *RTMP) onC1() error {
	var b [4 + 4 + 1528]byte
	_, err := r.read(b[0:])
	if nil != err {
		return err
	}

	timestamp := bin.BigEndian.Uint32(b[0:4])
	zero := bin.BigEndian.Uint32(b[4:8])
	if 0 != zero {
		str := fmt.Sprintf("Not zero of zero field(%d)", zero)
		r.event(EV_ZERO_ERROR, nil)
		fmt.Printf(str)
		return errors.New(str)
	}

	r.tsC1 = timestamp
	r.tsReadC1 = uint32(time.Now().Unix())

	bin.BigEndian.PutUint32(b[0:4], r.tsC1)
	bin.BigEndian.PutUint32(b[4:8], r.tsReadC1)

	r.write(b[0:])
	r.event(EV_C1, nil)
	r.stat = HANDSHAKE_C2

	return nil
}

func (r *RTMP) onC2() error {
	var b [4 + 4 + 1528]byte
	_, err := r.read(b[0:])
	if nil != err {
		return err
	}

	tsS1 := bin.BigEndian.Uint32(b[0:4])
	if tsS1 != r.tsS1 {
		str := fmt.Sprintf("Server timestamp (%d ? %d)\n", r.tsS1, tsS1)
		r.event(EV_MISMATCH_TS, nil)
		return errors.New(str)
	}

	r.event(EV_C2, nil)
	r.stat = STREAMING

	return nil
}

func (r *RTMP) read2BytesStreamID() (int, error) {
	var b [1]byte
	_, err := r.read(b[0:])
	if nil != err {
		return -1, err
	}
	return int(b[0] + 64), nil
}

func (r *RTMP) read3BytesStreamID() (int, error) {
	var b [2]byte
	_, err := r.read(b[0:])
	if nil != err {
		return -1, err
	}
	return int(b[0])*256 + int(b[1]) + 64, nil
}

func (r *RTMP) readStreamID(streamType uint8) (int, error) {
	switch streamType {
	case 0:
		return r.read2BytesStreamID()
	case 1:
		return r.read3BytesStreamID()
	default:
		return int(streamType), nil
	}
}

func (r *RTMP) readHeader(st *stream, headerType uint8) error {
	var b [MaxHeaderSize]byte

	headerLen := 0
	switch headerType {
	case 0:
		headerLen = 11
		st.pkt.absTimestamp = true
		break
	case 1:
		headerLen = 7
		break
	case 2:
		headerLen = 3
		break
	case 3:
		break
	default:
		return errors.New("Unknow header type")
	}

	_, err := r.read(b[0:headerLen])
	if nil != err {
		return err
	}

	switch headerType {
	case 0:
		st.pkt.msgStreamID = bin.LittleEndian.Uint32(b[7:headerLen])
		fallthrough
	case 1:
		st.pkt.msgLen = int(b[3]<<16 | b[4]<<8 | b[5])
		st.pkt.msgTypeID = b[6]
		fallthrough
	case 2:
		st.pkt.timestamp = uint32(b[0]<<16 | b[1]<<8 | b[2])
		fallthrough
	case 3:
		break
	}

	if 0xffffff == st.pkt.timestamp {
		_, err = r.read(b[0:4])
		if nil != err {
			return err
		}
		st.pkt.extTimestamp = true
		st.pkt.extTimestampVal = bin.BigEndian.Uint32(b[0:4])
	}

	return nil
}

func (r *RTMP) findStreamByID(streamID int) *stream {
	for _, st := range r.streams {
		if st.streamID == streamID {
			return st
		}
	}
	return nil
}

func (r *RTMP) onChunkSize(st *stream) error {
	var b [4]byte
	_, err := r.read(b[0:])
	if nil != err {
		return err
	}
	if 0 == (0x80 & b[0]) {
		str := fmt.Sprintf("First bit of chunk size mismatched")
		fmt.Printf(str)
		return errors.New(str)
	}

	r.inChunkSize = int(bin.BigEndian.Uint32(b[0:4]))
	return nil
}

func (r *RTMP) onPacket(st *stream) error {
	var err error
	switch st.pkt.msgTypeID {
	case MsgChunkSize:
		err = r.onChunkSize(st)
	case MsgStreamCancel:
	case MsgSendWndSize:
	case MsgRecvWndSize:
	case MsgBandwidth:
	case MsgCmdAMF0:
	case MsgCmdAMF3:
	case MsgInfoAMF0:
	case MsgInfoAMF3:
	case MsgShareObjAMF0:
	case MsgShareObjAMF3:
	case MsgAudio:
	case MsgVideo:
	case MsgAggregated:
	}

	st.pkt.body = nil
	st.pkt.msgLen = 0
	st.pkt.msgRead = 0

	return err
}

func (r *RTMP) onChunkBody(st *stream) error {
	if 0 == st.pkt.msgLen {
		return nil
	}

	if 0 < st.pkt.msgLen && nil == st.pkt.body {
		st.pkt.body = make([]byte, st.pkt.msgLen)
		st.pkt.msgRead = 0
	}

	nToRead := st.pkt.msgLen - st.pkt.msgRead
	nRead := r.inChunkSize
	if nToRead < nRead {
		nRead = nToRead
	}
	_, err := r.read(st.pkt.body[st.pkt.msgRead : st.pkt.msgRead+nRead])
	if nil != err {
		return err
	}
	st.pkt.msgRead += nRead

	if st.pkt.msgLen != st.pkt.msgRead {
		return nil
	}

	return r.onPacket(st)

}

func (r *RTMP) onChunk() error {
	var b [1]byte
	_, err := r.read(b[0:])
	if nil != err {
		return err
	}

	headerType := (b[0] & 0xc0) >> 6
	streamType := b[0] & 0x3f
	streamID, err := r.readStreamID(streamType)
	if nil != err {
		return err
	}

	st := r.findStreamByID(streamID)
	if nil == st {
		st = &stream{streamID: streamID}
		r.streams = append(r.streams, st)
		r.event(EV_NEW_STREAM, nil)
	}

	err = r.readHeader(st, headerType)
	if nil != err {
		return err
	}

	return r.onChunkBody(st)
}

func (r *RTMP) OnChunk() error {
	switch r.stat {
	case HANDSHAKE_C0:
		return r.onC0()
	case HANDSHAKE_C1:
		return r.onC1()
	case HANDSHAKE_C2:
		return r.onC2()
	case STREAMING:
		return r.onChunk()
	}
	return nil
}

func (r *RTMP) Write(b []byte) (int, error) {
	return r.conn.Write(b)
}

func (r *RTMP) Address() string {
	return r.conn.RemoteAddr().String()
}

func (r *RTMP) Connect() error {
	return nil
}

func (r *RTMP) Destroy() {

}
