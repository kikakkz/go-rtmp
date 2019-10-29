package flv

import (
	"errors"
)

const (
	TagVideo  = 8
	TagAudio  = 9
	TagScript = 18
)

const (
	TagHeaderSize = 11
	FLVHeaderSize = 9
)

type FLVTag struct {
	Tag               uint8
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
// FLVTag array is tags in the buffer
// int return consumed bytes
func Decode(buf []byte) ([]byte, []*FLVTag, int, error) {
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

	return header, tags, offs, nil
}

// Input buffer is raw data [AudioTag, VideoTag, ScriptTag]
func (flv *FLV) Encode(buf []byte, tagType int) (*FLVTag, error) {
	return nil, nil
}

func DecodeTag(buf []byte) (*FLVTag, int, error) {
	var tag FLVTag

	if len(buf) < 11 {
		return nil, 0, errors.New("Not full header")
	}

	tag.Tag = buf[0]
	tag.DataSize = int(buf[1]<<16 + buf[2]<<8 + buf[3])
	if len(buf) < TagHeaderSize+tag.DataSize {
		return nil, 0, errors.New("Not enough data")
	}

	tag.Timestamp = uint32(buf[4]<<16 + buf[5]<<8 + buf[6])
	tag.TimestampExtended = buf[7]
	tag.TagBuf = buf[0 : TagHeaderSize+tag.DataSize]
	tag.Data = buf[TagHeaderSize : TagHeaderSize+tag.DataSize]

	return &tag, TagHeaderSize + tag.DataSize, nil
}

func NewFLVTag(dataSize int, tagType uint8) *FLVTag {
	buf := make([]byte, dataSize+TagHeaderSize)
	return &FLVTag{Tag: tagType, DataSize: dataSize, TagBuf: buf, Data: buf[TagHeaderSize:]}
}

func (tag *FLVTag) Prepare() error {
	return nil
}
