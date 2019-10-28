package main

import (
	"fmt"
	rtmp "gortmp"
	"net"
)

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

	for {
		err := rtmpConn.OnChunk()
		if nil != err {
			fmt.Printf("%s\n", err)
			return
		}
		if playing {
			data := make([]byte, 256)
			err := rtmpConn.SendData(data[0:], rtmp.DataTypeAudio)
			if nil != err {
				// TODO: finish RTMP here due to network error
			}
		}
	}
}

func main() {
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
