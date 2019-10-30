package flv

import (
	bin "encoding/binary"
	"errors"
	"fmt"
)

const (
	TagAudio  = 8
	TagVideo  = 9
	TagScript = 18
)

const (
	TagHeaderSize = 11
	FLVHeaderSize = 9
)

type FLVTag struct {
	Tag               uint8
	PrevTagSize       uint32
	DataSize          int    /* 24 bits*/
	Timestamp         uint32 /* 24 bits */
	TimestampExtended uint8
	StreamID          uint32 /* Always 0, 24 bits */
	TagBuf            []byte /* Include header */
	Data              []byte /* Data only */
}

type AudioTag struct {
}

type VideoTag struct {
}

type MetadataTag struct {
}

type FLV struct {
}

func NewEncoder() *FLV {
	return &FLV{}
}

// Input buffer include tag header
// []byte is the header, nil if not header
// []byte is last tag size int this buf
// FLVTag array is tags in the buffer
// int return consumed bytes
func Decode(buf []byte) ([]byte, []byte, []*FLVTag, int, error) {
	var header []byte = nil
	var offs = 0

	if 'F' == buf[0] && 'L' == buf[1] && 'V' == buf[2] {
		header = buf[0:FLVHeaderSize]
		offs += FLVHeaderSize
	}

	var tags []*FLVTag = make([]*FLVTag, 0)

	for offs < len(buf) {
		tag, n, err := DecodeTag(buf[offs:])
		if nil == err {
			tags = append(tags, tag)
			offs += n
		} else {
			break
		}
	}

	var tailTagSize []byte = nil
	if offs+4 <= len(buf) {
		tailTagSize = buf[offs : offs+4]
		fmt.Printf("Tail tag size %x\n", tailTagSize)
	}

	return header, tailTagSize, tags, offs, nil
}

// Input buffer is raw data [AudioTag, VideoTag, ScriptTag]
func (flv *FLV) Encode(buf []byte, tagType int) (*FLVTag, error) {
	return nil, nil
}

func DecodeTag(buf []byte) (*FLVTag, int, error) {
	var tag FLVTag

	if len(buf) < TagHeaderSize+4 {
		return nil, 0, errors.New("Not full header")
	}

	tag.PrevTagSize = bin.BigEndian.Uint32(buf[0:4])
	tag.Tag = buf[4]
	tag.DataSize = int(bin.BigEndian.Uint32(buf[4:8]) & 0x00ffffff)
	if len(buf) < TagHeaderSize+tag.DataSize+4 {
		return nil, 0, errors.New("Not enough data")
	}

	tag.Timestamp = uint32(bin.BigEndian.Uint32(buf[8:12])&0xffffff00) >> 8
	tag.TimestampExtended = buf[11]

	tag.TagBuf = buf[0 : TagHeaderSize+tag.DataSize+4]
	tag.Data = buf[TagHeaderSize+4 : TagHeaderSize+tag.DataSize+4]

	return &tag, TagHeaderSize + tag.DataSize + 4, nil
}

func NewFLVTag(dataSize int, tagType uint8) *FLVTag {
	buf := make([]byte, dataSize+TagHeaderSize)
	return &FLVTag{Tag: tagType, DataSize: dataSize, TagBuf: buf, Data: buf[TagHeaderSize:]}
}

func (tag *FLVTag) Prepare() error {
	return nil
}
