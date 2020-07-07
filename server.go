package websocket

import (
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"io"
	"net"

	"regexp"
)

//Listener represents the websocket server
type Listener struct {
	net.Listener
}

//Conns saves pointers of Conn
type Conns []*Conn

//Connections save all the alive connections in an array
var Connections Conns

func checkHandShake(hs string) (string, error) {
	//read rfc6455 for precise define
	regex := regexp.MustCompile(`GET\s\/\w*\sHTTP\/\d\.\d`)
	if !regex.MatchString(hs) {
		return "", errors.New("Invalid or missing: [http Request] field")
	}
	regex = regexp.MustCompile(`Host:\s(\d{1,3}(\.\d{1,3}){3})|(\w*):\d{1,5}`)
	if !regex.MatchString(hs) {
		return "", errors.New("Invalid or missing: [Host] field")
	}
	regex = regexp.MustCompile(`Upgrade:\swebsocket`)
	if !regex.MatchString(hs) {
		return "", errors.New("Invalid or missing: [Upgrade] field")
	}
	regex = regexp.MustCompile(`Connection:\sUpgrade`)
	if !regex.MatchString(hs) {
		return "", errors.New("Invalid or missing: [Connection] field")
	}
	regex = regexp.MustCompile(`Sec-WebSocket-Version:\s13`)
	if !regex.MatchString(hs) {
		return "", errors.New("Invalid or missing: [Sec-WebSocket-Version] field")
	}
	regex = regexp.MustCompile(`Sec-WebSocket-Key:\s(.*)`)
	if !regex.MatchString(hs) {
		return "", errors.New("Invalid or missing: [Sec-WebSocket-Key] field")
	}
	res := regex.FindStringSubmatch(hs)[1]
	return res, nil
}

//DoHandShake response the handshake
func doHandShake(incoming string, w io.Writer) error {
	s, err := checkHandShake(incoming)
	if err != nil {
		return err
	}
	response := ""
	response += "HTTP/1.1 101 Switching Protocols\r\n"
	response += "Upgrade: websocket\r\n"
	response += "Connection: Upgrade\r\n"
	s = s[:len(s)-1] //fiilter the end of Line
	key := acceptKey(s)
	response += "Sec-WebSocket-Accept: " + key + "\r\n\r\n"
	_, e := w.Write([]byte(response))
	if e != nil {
		return e
	}
	return nil
}

func acceptKey(s string) string {
	s = s + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	hash := sha1.New()
	hash.Write([]byte(s))
	b := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(b)
}

//Listen returns websocket listener
func Listen(args ...interface{}) (l Listener, e error) {
	var listener net.Listener
	if len(args) > 3 {
		return Listener{nil}, errors.New("websocket listen: Too many arguments")
	}
	if args[0] == "tls" && len(args) == 3 {
		listener, e = tls.Listen(args[0].(string), args[1].(string), args[2].(*tls.Config))
	} else if args[0] == "tls" && len(args) != 3 {
		return Listener{nil}, errors.New("websocket listen: need 3 arguments")
	} else {
		listener, e = net.Listen("tcp", net.JoinHostPort(args[0].(string), args[1].(string)))
	}
	l.Listener = listener
	return
}

//Accept receive tcp connects and transform them to websocket connects
func (l *Listener) Accept() (conn Conn, e error) {
	c, _ := l.Listener.Accept()
	b := make([]byte, 600)
	n, _ := c.Read(b)
	b = b[:n]
	err := doHandShake(string(b), c)
	if err != nil {
		return Conn{nil, ""}, err
	}
	conn.Conn = c
	conn.typ = "server" //websocket in server-side
	Connections = append(Connections, &conn)
	return
}

//Broadcast will transport the bytes to all alive Conns
func Broadcast(buf interface{}) {
	for i := 0; i < len(Connections); i++ {
		var e error
		switch buf.(type) {
		case []byte:
			e = Connections[i].SendBinary(buf.([]byte))
		case string:
			e = Connections[i].SendText(buf.(string))
		}
		if e != nil { //remove the go-away Conn
			Connections[i].Close()
			Connections = append(Connections[:i], Connections[i+1:]...)
		}
	}
}
