package main

import (
	"flag"
	"fmt"
	flv "goflv"
	rtmp "gortmp"
	"net"
	"os"
)

var flvFilePath string

func init() {
	flag.StringVar(&flvFilePath, "flv", "./test.flv", "`flv` to be send")
}

func handleConnection(conn *net.TCPConn) {
	var rtmpConn *rtmp.RTMP
	playing := false

	rtmpConn = rtmp.New(conn, func(ev int, arg interface{}, b []byte) error {
		switch ev {
		case rtmp.EV_C0:
			fmt.Printf("[%s] C0 ---\n", rtmpConn.Address())
			break
		case rtmp.EV_READ_FAIL:
			fmt.Printf("[%s] READ FAIL ---\n", rtmpConn.Address())
			break
		case rtmp.EV_MISMATCH_VERSION:
			fmt.Printf("[%s] MISMATCH VERSION ---\n", rtmpConn.Address())
			break
		case rtmp.EV_C1:
			fmt.Printf("[%s] C1 ---\n", rtmpConn.Address())
			break
		case rtmp.EV_C2:
			fmt.Printf("[%s] C2 ---\n", rtmpConn.Address())
			break
		case rtmp.EV_ZERO_ERROR:
			fmt.Printf("[%s] ZERO ERROR ---\n", rtmpConn.Address())
			break
		case rtmp.EV_MISMATCH_TS:
			fmt.Printf("[%s] MISMATCH TIMESTAMP ---\n", rtmpConn.Address())
			break
		case rtmp.EV_NEW_STREAM:
			fmt.Printf("[%s] NEW STREAM(%d) ---\n", rtmpConn.Address(), arg.(int))
			break
		case rtmp.EV_DEL_STREAM:
			fmt.Printf("[%s] DEL STREAM(%d) ---\n", rtmpConn.Address(), arg.(int))
			playing = false
			break
		case rtmp.EV_CREATE_STREAM:
			fmt.Printf("[%s] CREATE STREAM(%d) ---\n", rtmpConn.Address(), arg.(int))
			break
		case rtmp.EV_START_PLAY:
			param := arg.(rtmp.PlayParam)
			fmt.Printf("[%s] PLAY(%s) ---\n", rtmpConn.Address(), param.ToString())
			playing = true
			break
		}
		return nil
	})
	defer rtmpConn.Destroy()

	flvFile, _ := os.Open(flvFilePath)
	defer flvFile.Close()
	testFile, _ := os.Create("./r.flv")
	defer testFile.Close()

	var buf [1024 * 1024]byte
	var offs = 0
	var tailTagSize []byte = nil
	var tags []*flv.FLVTag = nil
	var err error = nil
	var consumedBytes = 0
	var header []byte = nil

	for {
		err = rtmpConn.OnChunk()
		if nil != err {
			fmt.Printf("%s\n", err)
			return
		}
		if playing {
			var done = false
			n, err := flvFile.Read(buf[offs:])
			if nil != err {
				done = true
			}
			if 0 < n {
				header, tailTagSize, tags, consumedBytes, err = flv.Decode(buf[0 : offs+n])
				if nil != header {
					fmt.Printf("FLV -- header(%x)\n", header)
					testFile.Write(header)
					rtmpConn.SendData(header, rtmp.DataTypeMetadata)
				}
				for _, tag := range tags {
					fmt.Printf("FLV -- Tag %x/%x/%x ", tag.PrevTagSize, tag.DataSize, tag.Tag)
					var dataType = rtmp.DataTypeVideo
					switch tag.Tag {
					case flv.TagAudio:
						fmt.Printf("(AudioTag)\n")
						dataType = rtmp.DataTypeAudio
					case flv.TagVideo:
						fmt.Printf("(VideoTag)\n")
						dataType = rtmp.DataTypeVideo
					case flv.TagScript:
						fmt.Printf("(MetadataTag)\n")
						dataType = rtmp.DataTypeMetadata
					default:
						fmt.Printf("(Unknow)\n")
						continue
					}
					testFile.Write(tag.TagBuf)
					err = rtmpConn.SendData(tag.TagBuf, dataType)
					if nil != err {
						// TODO: finish RTMP here due to network error
					}
				}
				copy(buf[0:], buf[consumedBytes:])
				offs += len(buf) - consumedBytes
			}
			if done {
				if nil != tailTagSize {
					testFile.Write(tailTagSize)
					rtmpConn.SendData(tailTagSize, rtmp.DataTypeMetadata)
				}
				return
			}
			header, tailTagSize, tags, consumedBytes, err = flv.Decode(buf[0:n])
		}
	}
}

func main() {
	flag.Parse()
	fmt.Printf("FLV file path %s\n", flvFilePath)

	addr, err := net.ResolveTCPAddr("tcp", ":1935")
	if nil != err {
		fmt.Printf("Fail resolve addr(%s)\n", err)
		return
	}

	l, err := net.ListenTCP("tcp", addr)
	if nil != err {
		fmt.Printf("Fail listen RTMP port(%s)\n", err)
		return
	}
	defer l.Close()

	fmt.Printf("Listen RTMP at %s\n", l.Addr().String())

	for {
		conn, err := l.AcceptTCP()
		if nil != err {
			fmt.Printf("Fail accept RTMP connection(%s)\n", err)
			continue
		}
		defer conn.Close()
		go handleConnection(conn)
	}
}
