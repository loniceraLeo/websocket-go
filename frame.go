//An inplementation of websocket for golang

package websocket

import (
	"math/rand"
	"time"
)

//CreateDataFrame defined in rfc6455
func createDataFrame(fin, masked, first bool, typ string, data []byte) []byte {
	switch typ {
	case "text":
		if first {
			return buildMeta(fin, masked, 1, data)
		}
		return buildMeta(fin, masked, 0, data)
	case "binary":
		if first {
			return buildMeta(fin, masked, 2, data)
		}
		return buildMeta(fin, masked, 0, data)
	}
	return []byte(nil)
}

//CreateCloseFrame defined in rfc6455
func createCloseFrame(masked bool, closeCode code, reason []byte) []byte {
	temp := make([]byte, 2)
	temp[0] = byte(closeCode >> 8)
	temp[1] = byte(closeCode % 256)
	payload := append(temp, reason...)
	return buildMeta(true, masked, 8, payload)
}

//CreateControlFrame defined in rfc6455
func createControlFrame(masked bool, typ string, data []byte) []byte {
	switch typ {
	case "ping":
		return buildMeta(true, masked, 9, data)
	case "pong":
		return buildMeta(true, masked, 10, data)
	}
	return []byte(nil)
}
func buildMeta(fin, masked bool, opcode byte, payload []byte) (meta []byte) {
	var (
		//The first two bytes of ma
		payloadLen, metaLen = len(payload), 2
	)
	if masked {
		metaLen += 4
	}
	switch {
	case payloadLen < 126:
		metaLen += 0
	case payloadLen >= 126 && payloadLen < 65536:
		metaLen += 2
	case payloadLen >= 65536:
		metaLen += 8
	}
	meta = make([]byte, metaLen)
	if fin {
		meta[0] = 128
	}
	meta[0] += opcode
	if masked {
		meta[1] = 128
	}
	if payloadLen < 126 {
		meta[1] += byte(payloadLen)
	} else if payloadLen < 65536 {
		meta[1] += 126
		meta[2] = byte(payloadLen >> 8)
		meta[3] = byte(payloadLen % 256)
	} else {
		meta[1] += 127
		front32, end32 := payloadLen>>32, payloadLen%(1<<32)
		temp := []int{front32 >> 16, front32 % 65536, end32 >> 16, end32 % 65536}
		for i := 0; i < 4; i++ {
			meta[2+i*2] = byte(temp[i] >> 8)
			meta[3+i*2] = byte(temp[i] % 256)
		}
	}
	if masked {
		rand.Seed(time.Now().UnixNano())
		for i := 0; i < 4; i++ {
			meta[metaLen-4+i] = byte(rand.Intn(256))
		}
		for i := 0; i < payloadLen; i++ {
			payload[i] ^= meta[metaLen-4+i%4]
		}
	}
	return append(meta, payload...)
}
