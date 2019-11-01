package rtmp

/*
#include <stdint.h>
#include <string.h>

#include "librtmp/amf.h"
#include "librtmp/log.h"

static inline char *encodeByte(char *buf, uint8_t val)
{
	*buf++ = val;
	return buf;
}
static inline int bufLen(char *start, char *end)
{
	return end - start;
}
static inline char *bufInc(char *buf, int inc)
{
	return buf + inc;
}
static inline char *amfEncodeString(char *enc, char *end, char *str)
{
	AVal val = {
		.av_val = str,
		.av_len = strlen(str)
	};
	return AMF_EncodeString(enc, end, &val);
}
static inline char *amfEncodeNamedString(char *enc, char *end, char *name, char *str)
{
	AVal av_name = {
		.av_val = name,
		.av_len = strlen(name)
	};
	AVal val = {
		.av_val = str,
		.av_len = strlen(str)
	};
	return AMF_EncodeNamedString(enc, end, &av_name, &val);
}
static inline char *amfEncodeNamedNumber(char *enc, char *end, char *name, double number)
{
	AVal av_name = {
		.av_val = name,
		.av_len = strlen(name)
	};
	return AMF_EncodeNamedNumber(enc, end, &av_name, number);
}
static inline char *amfEncodeVersion(char *enc, char *end)
{
	AMFObjectProperty prop;
	AMFObjectProperty op;
	AMFObject obj;

#define STR2AVAL(av,str)	av.av_val = str; av.av_len = strlen(av.av_val)

	STR2AVAL(prop.p_name, "version");
	STR2AVAL(prop.p_vu.p_aval, "3,5,1,525");
	prop.p_type = AMF_STRING;
	obj.o_num = 1;
	obj.o_props = &prop;
	op.p_type = AMF_OBJECT;
	STR2AVAL(op.p_name, "data");
	op.p_vu.p_object = obj;
	return AMFProp_Encode(&op, enc, end);
}
*/
import "C"

import (
	bin "encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"
	"unsafe"
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
	MsgUserCtrl     = 4
	MsgWndAckSize   = 5
	MsgBandwidth    = 6
	MsgCmdAMF0      = 20
	MsgCmdAMF3      = 17
	MsgMetaAMF0     = 18
	MsgMetaAMF3     = 15
	MsgShareObjAMF0 = 19
	MsgShareObjAMF3 = 16
	MsgAudio        = 8
	MsgVideo        = 9
	MsgAggregated   = 22
)

const (
	StreamMsgCtrl = 0
	StreamMsgData = 1
)

const (
	StreamChunkCtrl = 2
)

const (
	CTRL_STREAM_BEGIN     = 0
	CTRL_STREAM_EOF       = 1
	CTRL_STREAM_DRY       = 2
	CTRL_SET_BUFFER_LEN   = 3
	CTRL_STREAM_IS_RECORD = 4
	CTRL_PING_REQUEST     = 6
	CTRL_PING_RESPONSE    = 7
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
	EV_CREATE_STREAM    = 10
	EV_START_PLAY       = 11
	EV_STOP_PLAY        = 11
)

type PlayParam struct {
	Playpath string
	App      string
	Reset    bool
}

type packet struct {
	headerType      uint8
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
	streamID     int
	extTimestamp bool
	pktIn        packet
	pktInCnt     int
	pktOut       *packet
	pktOutCnt    int
}

const (
	LINK_AUTH = 0x0001
	LINK_LIVE = 0x0002
	LINK_SWFV = 0x0004
	LINK_PLST = 0x0008
	LINK_BUFX = 0x0010
	LINK_FTCU = 0x0020
	LINK_FAPU = 0x0040
)

const (
	DataTypeMetadata = 0
	DataTypeVideo    = 1
	DataTypeAudio    = 2
)

const (
	ExtraStart = 3
)

// Not support crypto yet
type Link struct {
	hostname       *C.AVal
	port           int
	socksHost      *C.AVal
	socksPort      int
	playpath0      *C.AVal
	playpath       *C.AVal
	tcUrl          *C.AVal
	swfUrl         *C.AVal
	pageUrl        *C.AVal
	app            *C.AVal
	auth           *C.AVal
	flashVer       *C.AVal
	suscribePath   *C.AVal
	usherToken     *C.AVal
	token          *C.AVal
	pubUser        *C.AVal
	pubPasswd      *C.AVal
	extras         []*C.AMFObject
	linkFlags      uint16
	swfAge         int
	protocol       int
	timeout        int
	proxyUsed      bool
	audioCodecs    int
	videoCodecs    int
	videoFunction  int
	objectEncoding int
}

type RTMP struct {
	stat           int
	conn           *net.TCPConn
	evl            func(ev int, arg interface{}, body []byte) error
	tsC1           uint32
	tsS1           uint32
	tsReadC1       uint32
	streams        []*stream /* Incoming streams */
	inChunkSize    int
	outChunkSize   int
	createStreamID int /* Request of create stream */
	link           Link
	playing        bool
	audioStream    *stream
	videoStream    *stream
	dataStream     *stream
	timeStart      uint32
}

func (p *PlayParam) ToString() string {
	return fmt.Sprintf("/%s/%s(R:%v)", p.App, p.Playpath, p.Reset)
}

func New(conn *net.TCPConn, evl func(ev int, arg interface{}, body []byte) error) *RTMP {
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
	r.createStreamID = 10

	return &r
}

func init() {
	C.RTMP_LogSetLevel(C.RTMP_LOGALL)
}

func (r *RTMP) write(b []byte) (int, error) {
	return r.conn.Write(b)
}

func (r *RTMP) read(b []byte) (int, error) {
	r.conn.SetReadDeadline(time.Now().Add(2 * time.Millisecond))
	n, err := r.conn.Read(b)
	if nil != err {
		if !r.netTimeout(err) {
			fmt.Printf("Fail read(%s)\n", err)
			r.event(EV_READ_FAIL, nil, nil)
			return 0, err
		}
	}
	return n, nil
}

func (r *RTMP) event(ev int, arg interface{}, b []byte) {
	if nil != r.evl {
		r.evl(ev, arg, b)
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
		r.event(EV_MISMATCH_VERSION, nil, nil)
		// return errors.New(str)
	}

	r.write(b[0:1])

	timenow := uint32(time.Now().Unix())
	bin.BigEndian.PutUint32(b[0:4], timenow)
	r.tsS1 = timenow

	r.write(b[0:])
	r.event(EV_C0, nil, nil)
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
		str := fmt.Sprintf("Not zero of zero field(%d)\n", zero)
		r.event(EV_ZERO_ERROR, nil, nil)
		fmt.Printf(str)
		// return errors.New(str)
	}

	r.tsC1 = timestamp
	r.tsReadC1 = uint32(time.Now().Unix())

	bin.BigEndian.PutUint32(b[0:4], r.tsC1)
	bin.BigEndian.PutUint32(b[4:8], r.tsReadC1)

	r.write(b[0:])
	r.event(EV_C1, nil, nil)
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
		fmt.Printf("Server timestamp (%d ? %d)\n", r.tsS1, tsS1)
		r.event(EV_MISMATCH_TS, nil, nil)
		// return errors.New(str)
	}

	r.event(EV_C2, nil, nil)
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

func (st *stream) streamIdBytes() int {
	if 319 < st.streamID {
		return 2
	}
	if 64 < st.streamID {
		return 1
	}
	return 0
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

func (pkt *packet) headerLen() int {
	switch pkt.headerType {
	case 0:
		return 11
	case 1:
		return 7
	case 2:
		return 3
	case 3:
		return 0
	}
	return -1
}

func (r *RTMP) readHeader(st *stream) error {
	var b [MaxHeaderSize]byte

	headerLen := st.pktIn.headerLen()
	if headerLen < 0 {
		return errors.New("Unknow header type")
	}

	switch st.pktIn.headerType {
	case 0:
		st.pktIn.absTimestamp = true
	}

	_, err := r.read(b[0:headerLen])
	if nil != err {
		return err
	}

	switch st.pktIn.headerType {
	case 0:
		st.pktIn.msgStreamID = bin.LittleEndian.Uint32(b[7:headerLen])
		fallthrough
	case 1:
		st.pktIn.msgLen = int(b[3])<<16 | int(b[4])<<8 | int(b[5])
		st.pktIn.msgTypeID = b[6]
		fallthrough
	case 2:
		st.pktIn.timestamp = uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
		fallthrough
	case 3:
		break
	}

	if 0xffffff == st.pktIn.timestamp {
		_, err = r.read(b[0:4])
		if nil != err {
			return err
		}
		st.pktIn.extTimestamp = true
		st.pktIn.extTimestampVal = bin.BigEndian.Uint32(b[0:4])
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
	if 0 == (0x80 & st.pktIn.body[0]) {
		str := fmt.Sprintf("First bit of chunk size mismatched")
		fmt.Printf(str)
		return errors.New(str)
	}

	r.inChunkSize = int(bin.BigEndian.Uint32(st.pktIn.body[0:4]))
	return nil
}

func (r *RTMP) sendPacket(st *stream, pkt *packet) (int, error) {
	if 3 < pkt.headerType {
		return -1, errors.New("Invalid header type")
	}

	var timestamp uint32
	var timestampLast uint32

	if 0 < st.pktOutCnt {
		if 1 == pkt.headerType &&
			st.pktOut.msgLen == pkt.msgLen &&
			st.pktOut.headerType == pkt.headerType {
			pkt.headerType = 2
		}

		if 2 == pkt.headerType &&
			st.pktOut.timestamp == pkt.timestamp &&
			st.pktOut.extTimestamp == pkt.extTimestamp &&
			st.pktOut.extTimestampVal == pkt.extTimestampVal {
			pkt.headerType = 3
			if pkt.extTimestamp {
				timestampLast = st.pktOut.extTimestampVal
			} else {
				timestampLast = st.pktOut.timestamp
			}
		}
	}

	headerLen := pkt.headerLen() + 1 // For header type
	streamIdBytes := st.streamIdBytes()
	headerLen += streamIdBytes
	timestamp = pkt.timestamp - timestampLast
	if 0xffffff < timestamp && st.extTimestamp {
		pkt.extTimestamp = true
		pkt.extTimestampVal = timestamp
		pkt.timestamp = 0xffffff
		headerLen += 4
	}

	var pktBuf []byte = nil
	var offs = MaxHeaderSize - headerLen

	if nil == pkt.body {
		pktBuf = make([]byte, headerLen+1)
	} else {
		pktBuf = pkt.body[offs : MaxHeaderSize+1]
	}

	end := (*C.char)(unsafe.Pointer(&pktBuf[MaxHeaderSize-offs]))
	enc := (*C.char)(unsafe.Pointer(&pktBuf[0]))
	start := enc

	val := pkt.headerType << 6
	switch streamIdBytes {
	case 0:
		val |= uint8(st.streamID)
	case 1:
	case 2:
		val |= 1
	}
	enc = C.encodeByte(enc, C.uint8_t(val))

	if 0 < streamIdBytes {
		enc = C.encodeByte(enc, C.uint8_t(st.streamID-64))
	}
	if 1 < streamIdBytes {
		enc = C.encodeByte(enc, C.uint8_t(st.streamID>>8))
	}

	if 1 < headerLen {
		enc = C.AMF_EncodeInt24(enc, end, C.int(pkt.timestamp))
	}
	if 4 < headerLen {
		enc = C.AMF_EncodeInt24(enc, end, C.int(pkt.msgLen))
		enc = C.encodeByte(enc, C.uint8_t(pkt.msgTypeID))
	}
	if 8 < headerLen {
		len := C.bufLen(start, enc)
		bin.LittleEndian.PutUint32(pktBuf[len:len+4], pkt.msgStreamID)
		enc = C.bufInc(enc, 4)
	}
	if pkt.extTimestamp {
		enc = C.AMF_EncodeInt32(enc, end, C.int(pkt.extTimestampVal))
	}

	sendBytes := 0
	sendBuf := pkt.body[offs : pkt.msgLen+MaxHeaderSize]
	offs = 0
	writeLen := r.outChunkSize
	nextWriteEnd := headerLen + writeLen

	for sendBytes < pkt.msgLen {
		if pkt.msgLen < sendBytes+r.outChunkSize {
			writeLen = pkt.msgLen - sendBytes
			nextWriteEnd = headerLen + pkt.msgLen
		}

		if 0 != offs {
			offs -= 1
			offs -= streamIdBytes
			if pkt.extTimestamp {
				offs -= 4
			}
			sendBuf[offs] = 0xc0 | val
			if 0 < streamIdBytes {
				sendBuf[offs+1] = byte(st.streamID - 64)
			}
			if 1 < streamIdBytes {
				sendBuf[offs+2] = byte(st.streamID >> 8)
			}
			if pkt.extTimestamp {
				end = (*C.char)(unsafe.Pointer(&sendBuf[offs+5+streamIdBytes]))
				enc = (*C.char)(unsafe.Pointer(&sendBuf[offs+1+streamIdBytes]))
				enc = C.AMF_EncodeInt32(enc, end, C.int(pkt.extTimestampVal))
				if nil == enc {
					fmt.Printf("Error: fail set ext timestamp\n")
				}
			}
		}

		n, err := r.write(sendBuf[offs:nextWriteEnd])
		if n != nextWriteEnd-offs {
			return n, errors.New("Not full wrote")
		}
		if nil != err {
			return n, err
		}

		sendBytes += writeLen
		offs = nextWriteEnd
		nextWriteEnd += r.outChunkSize
	}

	st.pktOut = pkt
	return pkt.msgLen, nil
}

func stringToCString(str string) *C.char {
	var localStrBytes = make([]byte, len(str)+1)
	copy(localStrBytes, []byte(str))
	return (*C.char)(unsafe.Pointer(&(localStrBytes[0])))
}

func obtainPacket(bodySize int, headerType uint8, msgTypeID uint8, msgStreamID uint32) *packet {
	var pkt packet
	pkt.body = make([]byte, bodySize+MaxHeaderSize+1)
	pkt.headerType = headerType
	pkt.msgTypeID = msgTypeID
	pkt.timestamp = 0
	pkt.msgStreamID = msgStreamID
	pkt.absTimestamp = false
	return &pkt
}

func (r *RTMP) onConnectResp(st *stream, txID int) error {
	pkt := obtainPacket(386, 1, MsgCmdAMF0, StreamMsgCtrl)

	end := (*C.char)(unsafe.Pointer(&pkt.body[len(pkt.body)-1]))
	enc := (*C.char)(unsafe.Pointer(&pkt.body[MaxHeaderSize]))
	start := enc

	enc = C.amfEncodeString(enc, end, stringToCString("_result"))
	enc = C.AMF_EncodeNumber(enc, end, C.double(txID))

	enc = C.encodeByte(enc, C.AMF_OBJECT)
	enc = C.amfEncodeNamedString(enc, end, stringToCString("fmsVer"), stringToCString("FMS/3,5,1,525"))
	enc = C.amfEncodeNamedNumber(enc, end, stringToCString("capabilities"), 31.0)
	enc = C.amfEncodeNamedNumber(enc, end, stringToCString("mode"), 1.0)
	enc = C.encodeByte(enc, 0)
	enc = C.encodeByte(enc, 0)
	enc = C.encodeByte(enc, C.AMF_OBJECT_END)

	enc = C.encodeByte(enc, C.AMF_OBJECT)
	enc = C.amfEncodeNamedString(enc, end, stringToCString("level"), stringToCString("status"))
	enc = C.amfEncodeNamedString(enc, end, stringToCString("code"), stringToCString("NetConnection.Connect.Success"))
	enc = C.amfEncodeNamedString(enc, end, stringToCString("description"), stringToCString("Connection success"))
	enc = C.amfEncodeNamedNumber(enc, end, stringToCString("objectEncoding"), C.double(r.link.objectEncoding))

	enc = C.amfEncodeVersion(enc, end)

	enc = C.encodeByte(enc, 0)
	enc = C.encodeByte(enc, 0)
	enc = C.encodeByte(enc, C.AMF_OBJECT_END)

	pkt.msgLen = int(C.bufLen(start, enc))

	_, err := r.sendPacket(st, pkt)
	return err
}

func (r *RTMP) sendChunkSize(st *stream) error {
	var ctrlStream = stream{streamID: StreamChunkCtrl, extTimestamp: true}

	pkt := obtainPacket(256, 0, MsgChunkSize, StreamMsgCtrl)
	end := (*C.char)(unsafe.Pointer(&pkt.body[len(pkt.body)-1]))
	enc := (*C.char)(unsafe.Pointer(&pkt.body[MaxHeaderSize]))
	start := enc

	enc = C.AMF_EncodeInt32(enc, end, C.int(r.outChunkSize))

	pkt.msgLen = int(C.bufLen(start, enc))

	_, err := r.sendPacket(&ctrlStream, pkt)
	return err
}

func (r *RTMP) sendBandwidth(st *stream, bw int) error {
	var ctrlStream = stream{streamID: StreamChunkCtrl, extTimestamp: true}

	pkt := obtainPacket(256, 0, MsgBandwidth, StreamMsgCtrl)
	end := (*C.char)(unsafe.Pointer(&pkt.body[len(pkt.body)-1]))
	enc := (*C.char)(unsafe.Pointer(&pkt.body[MaxHeaderSize]))
	start := enc

	enc = C.AMF_EncodeInt32(enc, end, C.int(bw))
	enc = C.encodeByte(enc, 2)

	pkt.msgLen = int(C.bufLen(start, enc))

	_, err := r.sendPacket(&ctrlStream, pkt)
	return err
}

func (r *RTMP) sendWndAckSize(st *stream, size int) error {
	var ctrlStream = stream{streamID: StreamChunkCtrl, extTimestamp: true}

	pkt := obtainPacket(256, 0, MsgWndAckSize, StreamMsgCtrl)
	end := (*C.char)(unsafe.Pointer(&pkt.body[len(pkt.body)-1]))
	enc := (*C.char)(unsafe.Pointer(&pkt.body[MaxHeaderSize]))
	start := enc

	enc = C.AMF_EncodeInt32(enc, end, C.int(size))

	pkt.msgLen = int(C.bufLen(start, enc))

	_, err := r.sendPacket(&ctrlStream, pkt)
	return err
}

func (r *RTMP) onConnect(st *stream, txID int, cmdObj *C.AMFObject, extraObjs []*C.AMFObject, obj *C.AMFObject) error {
	count := int(C.AMF_CountProp(cmdObj))
	for i := 0; i < count; i++ {
		name := C.AVal{av_val: nil, av_len: 0}
		valStr := C.AVal{av_val: nil, av_len: 0}

		C.AMFProp_GetName(C.AMF_GetProp(cmdObj, nil, C.int(i)), &name)
		C.AMFProp_GetString(C.AMF_GetProp(cmdObj, nil, C.int(i)), &valStr)
		valBool := (0 != C.AMFProp_GetBoolean(C.AMF_GetProp(cmdObj, nil, C.int(i))))
		valInt := int(C.AMFProp_GetNumber(C.AMF_GetProp(cmdObj, nil, C.int(i))))
		prop := C.GoStringN(name.av_val, name.av_len)

		switch prop {
		case "app":
			r.link.app = &valStr
		case "flashver":
			r.link.flashVer = &valStr
		case "swfUrl":
			r.link.swfUrl = &valStr
		case "tcUrl":
			r.link.tcUrl = &valStr
		case "pageUrl":
			r.link.pageUrl = &valStr
		case "fpad":
			r.link.proxyUsed = valBool
		case "audioCodecs":
			r.link.audioCodecs = valInt
		case "videoCodecs":
			r.link.videoCodecs = valInt
		case "videoFunction":
			r.link.videoFunction = valInt
		case "objectEncoding":
			r.link.objectEncoding = valInt
		}
	}

	if nil != extraObjs {
		r.link.extras = extraObjs
	}

	r.sendWndAckSize(st, 5000000)
	r.sendBandwidth(st, 5000000)
	r.sendCtrl(st, CTRL_STREAM_BEGIN, nil)

	return r.onConnectResp(st, txID)
}

func (r *RTMP) onCreateStreamResp(st *stream, txID int, streamID int) error {
	pkt := obtainPacket(256, 1, MsgCmdAMF0, StreamMsgCtrl)

	end := (*C.char)(unsafe.Pointer(&pkt.body[len(pkt.body)-1]))
	enc := (*C.char)(unsafe.Pointer(&pkt.body[MaxHeaderSize]))
	start := enc

	enc = C.amfEncodeString(enc, end, stringToCString("_result"))
	enc = C.AMF_EncodeNumber(enc, end, C.double(txID))
	enc = C.encodeByte(enc, C.AMF_NULL)
	enc = C.AMF_EncodeNumber(enc, end, C.double(streamID))

	pkt.msgLen = int(C.bufLen(start, enc))

	_, err := r.sendPacket(st, pkt)
	return err
}

func (r *RTMP) onDeleteStream(st *stream, txID int, cmdObj *C.AMFObject, extraObjs []*C.AMFObject, obj *C.AMFObject) error {
	streamID := C.AMFProp_GetNumber(C.AMF_GetProp(obj, nil, 3))
	for k, st := range r.streams {
		if int(streamID) == st.streamID {
			r.streams = append(r.streams[0:k], r.streams[k+1:]...)
		}
	}
	r.playing = false
	r.event(EV_DEL_STREAM, int(streamID), nil)
	return nil
}

func (r *RTMP) onCreateStream(st *stream, txID int, cmdObj *C.AMFObject, extraObjs []*C.AMFObject, obj *C.AMFObject) error {
	streamID := r.createStreamID
	r.createStreamID += 1
	r.event(EV_CREATE_STREAM, streamID, nil)

	r.sendChunkSize(st)
	r.sendCtrl(st, CTRL_STREAM_BEGIN, nil)

	return r.onCreateStreamResp(st, txID, streamID)
}

func (r *RTMP) sendCtrl(st *stream, ctrl int, param interface{}) error {
	var ctrlStream = stream{streamID: StreamChunkCtrl, extTimestamp: true}

	pkt := obtainPacket(256, 0, MsgUserCtrl, StreamMsgCtrl)
	end := (*C.char)(unsafe.Pointer(&pkt.body[len(pkt.body)-1]))
	enc := (*C.char)(unsafe.Pointer(&pkt.body[MaxHeaderSize]))
	start := enc

	enc = C.AMF_EncodeInt16(enc, end, C.short(ctrl))

	switch ctrl {
	case CTRL_STREAM_BEGIN:
		enc = C.AMF_EncodeInt32(enc, end, C.int(st.streamID))
	}

	pkt.msgLen = int(C.bufLen(start, enc))

	_, err := r.sendPacket(&ctrlStream, pkt)
	return err
}

func (r *RTMP) sendPlayResp(st *stream, txID int, reset bool) error {
	pkt := obtainPacket(512, 1, MsgCmdAMF0, StreamMsgCtrl)

	end := (*C.char)(unsafe.Pointer(&pkt.body[len(pkt.body)-1]))
	enc := (*C.char)(unsafe.Pointer(&pkt.body[MaxHeaderSize]))
	start := enc

	enc = C.amfEncodeString(enc, end, stringToCString("onStatus"))
	enc = C.AMF_EncodeNumber(enc, end, C.double(0))

	enc = C.encodeByte(enc, C.AMF_NULL)

	enc = C.encodeByte(enc, C.AMF_OBJECT)

	enc = C.amfEncodeNamedString(enc, end, stringToCString("level"), stringToCString("status"))
	enc = C.amfEncodeNamedString(enc, end, stringToCString("code"), stringToCString("NetStream.Play.Start"))
	enc = C.amfEncodeNamedString(enc, end, stringToCString("description"), stringToCString("Start play"))
	enc = C.amfEncodeNamedString(enc, end, stringToCString("details"), r.link.playpath.av_val)
	enc = C.amfEncodeNamedString(enc, end, stringToCString("clientid"), stringToCString("clientid"))
	enc = C.encodeByte(enc, 0)
	enc = C.encodeByte(enc, 0)
	enc = C.encodeByte(enc, C.AMF_OBJECT_END)

	pkt.msgLen = int(C.bufLen(start, enc))

	_, err := r.sendPacket(st, pkt)
	return err
}

func (r *RTMP) onPlayResp(st *stream, txID int, reset bool) error {
	r.sendCtrl(st, CTRL_STREAM_BEGIN, nil)
	return r.sendPlayResp(st, txID, reset)
}

func (r *RTMP) onPlay(st *stream, txID int, cmdObj *C.AMFObject, extraObjs []*C.AMFObject, obj *C.AMFObject) error {
	var val C.AVal
	C.AMFProp_GetString(C.AMF_GetProp(obj, nil, 3), &val)
	if 0 == val.av_len {
		return errors.New("Invalid playpath")
	}

	r.link.playpath = &val

	startTime := int64(C.AMFProp_GetNumber(C.AMF_GetProp(obj, nil, 4)))
	duration := int64(C.AMFProp_GetNumber(C.AMF_GetProp(obj, nil, 5)))
	reset := (0 != C.AMFProp_GetNumber(C.AMF_GetProp(obj, nil, 6)))

	fmt.Printf("PLAY %s(%d/%d need reset %v)\n", C.GoString(val.av_val), startTime, duration, reset)

	r.event(EV_START_PLAY, PlayParam{
		Playpath: C.GoString(val.av_val),
		App:      C.GoString(r.link.app.av_val),
		Reset:    reset}, nil)

	r.audioStream = &stream{streamID: r.createStreamID, extTimestamp: true}
	r.sendCtrl(r.audioStream, CTRL_STREAM_BEGIN, nil)
	r.createStreamID += 1
	r.videoStream = &stream{streamID: r.createStreamID, extTimestamp: true}
	r.sendCtrl(r.videoStream, CTRL_STREAM_BEGIN, nil)
	r.createStreamID += 1
	r.dataStream = &stream{streamID: r.createStreamID, extTimestamp: false}
	r.sendCtrl(r.dataStream, CTRL_STREAM_BEGIN, nil)

	r.playing = true
	r.timeStart = uint32(time.Now().Unix())

	return r.onPlayResp(st, txID, reset)
}

func (r *RTMP) onCmd(st *stream, obj *C.AMFObject) error {
	var aval C.AVal
	var cmdObj C.AMFObject
	var extraObjs []*C.AMFObject = nil

	C.AMFProp_GetString(C.AMF_GetProp(obj, nil, 0), &aval)
	txID := int(C.AMFProp_GetNumber(C.AMF_GetProp(obj, nil, 1)))
	C.AMFProp_GetObject(C.AMF_GetProp(obj, nil, 2), &cmdObj)
	count := int(C.AMF_CountProp(obj))

	if ExtraStart < count {
		extraObjs = make([]*C.AMFObject, 0)
		for i := ExtraStart; i < count; i++ {
			var extObj C.AMFObject
			C.AMFProp_GetObject(C.AMF_GetProp(obj, nil, C.int(i)), &extObj)
			extraObjs = append(extraObjs, &extObj)
		}
	}

	cmd := C.GoString(aval.av_val)
	fmt.Printf("CMD --- %s(tx: %d)\n", cmd, txID)

	switch cmd {
	case "connect":
		return r.onConnect(st, txID, &cmdObj, extraObjs, obj)
	case "createStream":
		return r.onCreateStream(st, txID, &cmdObj, extraObjs, obj)
	case "deleteStream":
		return r.onDeleteStream(st, txID, &cmdObj, extraObjs, obj)
	case "play":
		return r.onPlay(st, txID, &cmdObj, extraObjs, obj)
	}

	return nil
}

func (r *RTMP) onCmdAMF0(st *stream) error {
	var obj C.AMFObject
	var rc C.int

	b := st.pktIn.body
	rc = C.AMF_Decode(&obj, (*C.char)(unsafe.Pointer(&b[0])), C.int(len(b)), 0)
	if rc < 0 {
		str := fmt.Sprintf("Fail decode command(%d)\n", rc)
		fmt.Printf(str)
		return errors.New(str)
	}
	C.AMF_Dump(&obj)

	return r.onCmd(st, &obj)
}

func (r *RTMP) onCmdAMF3(st *stream) error {
	var obj C.AMFObject
	var rc C.int

	b := st.pktIn.body
	rc = C.AMF_Decode(&obj, (*C.char)(unsafe.Pointer(&b[1])), C.int(len(b)-1), 0)
	if rc < 0 {
		str := fmt.Sprintf("Fail decode command(%d)\n", rc)
		fmt.Printf(str)
		return errors.New(str)
	}
	C.AMF_Dump(&obj)

	return r.onCmd(st, &obj)
}

func (r *RTMP) onPacket(st *stream) error {
	var err error

	switch st.pktIn.msgTypeID {
	case MsgChunkSize:
		err = r.onChunkSize(st)
	case MsgStreamCancel:
	case MsgSendWndSize:
	case MsgWndAckSize:
	case MsgBandwidth:
	case MsgCmdAMF0:
		err = r.onCmdAMF0(st)
	case MsgCmdAMF3:
		err = r.onCmdAMF3(st)
	case MsgMetaAMF0:
	case MsgMetaAMF3:
	case MsgShareObjAMF0:
	case MsgShareObjAMF3:
	case MsgAudio:
	case MsgVideo:
	case MsgAggregated:
	}

	return err
}

func (r *RTMP) onChunkBody(st *stream) error {
	if 0 == st.pktIn.msgLen {
		return nil
	}

	if 0 < st.pktIn.msgLen && nil == st.pktIn.body {
		st.pktIn.body = make([]byte, st.pktIn.msgLen)
		st.pktIn.msgRead = 0
	}

	nToRead := st.pktIn.msgLen - st.pktIn.msgRead
	nRead := r.inChunkSize
	if nToRead < nRead {
		nRead = nToRead
	}

	var err error = nil
	var dataLen = 0
	var dataTotal = 0
	var readStart = st.pktIn.msgRead

	if 0 < nRead {
		for dataTotal < nRead {
			dataLen, err = r.read(st.pktIn.body[st.pktIn.msgRead : readStart+nRead])
			if nil != err {
				return err
			}
			dataTotal += dataLen
			st.pktIn.msgRead += dataLen
		}
	}

	if st.pktIn.msgLen != st.pktIn.msgRead {
		return nil
	}

	err = r.onPacket(st)

	st.pktIn.body = nil
	st.pktIn.msgLen = 0
	st.pktIn.msgRead = 0

	return err
}

func (r *RTMP) SendData(data []byte, dataType int, timeDelta int) error {
	if !r.playing {
		return errors.New("Not playing")
	}

	var pkt *packet
	var st *stream

	switch dataType {
	case DataTypeAudio:
		pkt = obtainPacket(len(data), 0, MsgAudio, StreamMsgData)
		st = r.audioStream
	case DataTypeVideo:
		pkt = obtainPacket(len(data), 0, MsgVideo, StreamMsgData)
		st = r.videoStream
	case DataTypeMetadata:
		pkt = obtainPacket(len(data), 0, MsgMetaAMF0, StreamMsgData)
		st = r.dataStream
	default:
		return errors.New("Invalid data type")
	}

	pkt.timestamp = r.timeStart + uint32(timeDelta)
	copy(pkt.body[MaxHeaderSize:], data[0:])
	pkt.msgLen = len(data)

	_, err := r.sendPacket(st, pkt)
	return err
}

func (r *RTMP) onChunk() error {
	var b [1]byte
	n, err := r.read(b[0:])
	if nil != err {
		return err
	}

	if 0 == n {
		return nil
	}

	streamType := b[0] & 0x3f
	streamID, err := r.readStreamID(streamType)
	if nil != err {
		return err
	}

	st := r.findStreamByID(streamID)
	if nil == st {
		st = &stream{streamID: streamID, extTimestamp: true}
		r.streams = append(r.streams, st)
		r.event(EV_NEW_STREAM, streamID, nil)
	}

	st.pktIn.headerType = (b[0] & 0xc0) >> 6
	err = r.readHeader(st)
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
