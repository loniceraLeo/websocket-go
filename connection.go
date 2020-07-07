//This package implementes both server-side and client-side(none browser) connection

package websocket

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	//"net/url"
)

//Four state of the connection
const (
	CONNECTING = iota
	OPENING
	CLOSING
	CLOSED
)

//The max bytes transported in one frame
const (
	MAXBUF = 512 //0.5KiB
)

//Conn is a websocket connect
type Conn struct {
	net.Conn
	typ string
}

//Client represents a websocket client
type Client struct {
	connect Conn
	state   byte
}

type code int16

var data []byte

//Read reads the network stream.Notice:If the type of the frame is text or
//binary, it will return the buffer after several calls(maybe once).But
//the application data in the ping-pong frame will never be sent to users.
//Indeed, they will be processed as soon as they reach.
func (conn *Conn) Read(p []byte) (d []byte, op byte, ok bool, err error) {
	//if fin=true, the function return <data, true>, otherwise it returns <nil, false>
	n, e := conn.Conn.Read(p)
	if e != nil {
		err = e
		return
	}
	p = p[:n]
	fin, opcode, buf := extraceFrame(p)
	//ALARM:we assume that during the transport, the opcode is same
	processBuffer(opcode, buf)
	if opcode == 8 {
		op = opcode
		err = errors.New("remote closed:" + string(data))
		data = nil
		return
	}
	if opcode == 9 {
		fmt.Println("ping")
		conn.SendPong(buf)
		return nil, 9, false, nil
	}
	if fin == true && (opcode == 1 || opcode == 2) {
		op = opcode
		d = make([]byte, len(data))
		copy(d, data)
		data = nil
		ok = true
	} else {
		d, ok = nil, false
	}
	return
}

//Read calls the underlying method os connect
func (client *Client) Read(p []byte) (d []byte, op byte, ok bool, err error) {
	d, op, ok, err = client.connect.Read(p)
	return
}

//Close closes the websocket connection
func (conn *Conn) Close() {
	conn.Conn.Close()
	conn = nil
}

//Close closes the connection to the server-side, it MUST BE called by a client
func (client *Client) Close(closeCode code, reason string) {
	client.state = CLOSING
	client.connect.sendCloseFrame(closeCode, []byte(reason))
	client.state = CLOSED
	client.connect.Close()
	client = nil
}

//SendText sends text message to the other side
//if the message is from server, it MUST NOT be masked
func (conn *Conn) SendText(str string) error {
	var b []byte
	for i := 0; i < len(str); i += MAXBUF {
		var fin, masked, first bool = false, false, false
		if i == 0 {
			first = true
		}
		if conn.typ == "client" {
			masked = true
		}
		if i+MAXBUF > len(str) {
			fin = true
		}
		b = createDataFrame(fin, masked, first, "text", []byte(str))
		_, err := conn.Conn.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
	/*r := CreateDataFrame(true, false, true, "text", []byte(str))
	conn.Conn.Write(r)*/
}

//SendText calls the underlying method of connect
func (client *Client) SendText(str string) error {
	e := client.connect.SendText(str)
	return e
}

//SendBinary sends bianry data to the other side
//if the data is from server, it MUST NOT be masked
func (conn *Conn) SendBinary(buf []byte) error {
	var b []byte
	for i := 0; i < len(buf); i += MAXBUF {
		var fin, masked, first bool = false, false, false
		if i == 0 {
			first = true
		}
		if conn.typ == "client" {
			masked = true
		}
		if i+MAXBUF > len(buf) {
			fin = true
		}
		b = createDataFrame(fin, masked, first, "binary", []byte(buf))
		_, err := conn.Conn.Write(b)
		if err != nil {
			return err
		}
	}
	return nil
}

//SendBinary calls the underlying method of connect
func (client *Client) SendBinary(buf []byte) error {
	e := client.connect.SendBinary(buf)
	return e
}

//SendCloseFrame sends the close frame to the other side.A close code is
//necessary,reason is optional.Please check the rfc6455 document for more information.
func (conn *Conn) sendCloseFrame(closeCode code, reason []byte) error {
	var (
		b      []byte
		masked bool = false
	)
	if conn.typ == "client" {
		masked = true
	}
	b = createCloseFrame(masked, closeCode, reason)
	_, err := conn.Conn.Write(b)
	if err != nil {
		return err
	}
	return nil
}

//SendPing sends a ping frame to the other side.
func (conn *Conn) SendPing(inf []byte) error {
	var (
		b      []byte
		masked bool = false
	)
	if conn.typ == "client" {
		masked = true
	}
	b = createControlFrame(masked, "ping", inf)
	_, err := conn.Conn.Write(b)
	if err != nil {
		return err
	}
	return nil
}

//SendPing calls the underlying method of connect
func (client *Client) SendPing(inf []byte) error {
	e := client.connect.SendPing(inf)
	return e
}

//SendPong sends a pong frame to the other side.The application data must
//equal with the data in ping frame
func (conn *Conn) SendPong(inf []byte) error {
	var (
		b      []byte
		masked bool = false
	)
	if conn.typ == "client" {
		masked = true
	}
	b = createControlFrame(masked, "pong", inf)
	_, err := conn.Conn.Write(b)
	if err != nil {
		return err
	}
	return nil
}

//SendPong calls the underlying method of connect
func (client *Client) SendPong(inf []byte) error {
	e := client.connect.SendPong(inf)
	return e
}

func extraceFrame(data []byte) (bool, byte, []byte) {
	var (
		fin, masked bool
		opcode      byte
		payloadLen  int
		start       int
		d           []byte
	)
	start = 2
	if data[0] >= 128 {
		fin = true
	} else {
		fin = false
	}
	opcode = data[0] % 16
	if data[1] >= 128 {
		masked = true
	} else {
		masked = false
	}
	t := int(data[1] % 128)
	switch true {
	case t < 126:
		payloadLen = t
	case t == 126:
		payloadLen = int(data[2])<<8 + int(data[3])
		start += 2
	case t == 127:
		for i := 0; i < 8; i++ {
			payloadLen += int(data[9-i]) << (8 * i)
		}
		start += 8
	}
	if masked == true {
		mask := make([]byte, 4)
		for i := 0; i < 4; i++ {
			mask[i] = data[start+i]
		}
		start += 4
		for i := start; i < len(data); i++ {
			data[i] ^= mask[(i-start)%4]
		}
	}
	d = data[start:]
	return fin, opcode, d
}

func processBuffer(opcode byte, buf []byte) {
	switch true {
	case opcode == 1 || opcode == 2: //text or binary
		data = append(data, buf...)
	case opcode == 8: //for example, it may look like "close:1009"
		s := "close code:"
		closeCode := int(buf[0])<<8 + int(buf[1])
		s += strconv.Itoa(closeCode)
		data = []byte(s)
	case opcode == 9 || opcode == 10: //ping-pong frames
	}
}

func parseURI(uri string) (host, port string, e error) {
	reg := regexp.MustCompile(`(ws|wss):\/\/(.*):(\d{1,5})`)
	if !reg.MatchString(uri) {
		e = errors.New("invalid uri format")
		return
	}
	s := reg.FindAllStringSubmatch(uri, -1)
	host, port = s[0][2], s[0][3]
	return
}

//Connect returns a client object and any error(if exists)
func Connect(uri string) (*Client, error) {
	host, port, err := parseURI(uri)
	if err != nil {
		return nil, err
	}
	client := new(Client)
	conn, e := net.Dial("tcp", net.JoinHostPort(host, port))
	if e != nil {
		return nil, e
	}
	client.connect.Conn = conn
	client.connect.typ = "client"
	client.state = CONNECTING

	req, check := createRequest(host, port)
	client.connect.Write([]byte(req))

	b := make([]byte, 400)
	n, _ := client.connect.Conn.Read(b)
	b = b[:n]
	bo := checkResponseKey(string(b), check)
	if bo == false {
		return nil, errors.New("Invalid [websocket-accept-key] value")
	}
	client.state = OPENING
	return client, nil
}

func checkResponseKey(str string, key string) (b bool) {
	b = strings.Contains(str, key)
	return
}

func createRequest(host, port string) (req string, check string) {
	b, _ := generateKey(16)
	key := base64.StdEncoding.EncodeToString(b)
	check = acceptKey(key)

	req = ""
	req += "GET / HTTP/1.1\r\n"
	req += "Host: " + host + ":" + port + "\r\n"
	req += "Upgrade: websocket\r\n"
	req += "Connection: Upgrade\r\n"
	req += "Sec-WebSocket-Key: " + key + "\r\n"
	req += "Sec-WebSocket-Version: 13\r\n\r\n"

	return req, check
}

func generateKey(length int) (b []byte, e error) {
	b = make([]byte, length)
	_, e = rand.Read(b)
	return
}
