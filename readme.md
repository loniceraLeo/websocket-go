# Websocket-go

This is a brand new implementation of websocket-protocol for golang.It encapsulates several functions to make things easier.

## Download

To get source code, run
`go get github.com/loniceraLeo/websocket-go`

## Usage

### Server-side

You can create a websocket-server in this way:  

`listener, e := websocket.Listen(host, port)`

Then the server will listen the incoming connections on this address.If you want to run the server on tls,you can call the function in this way:  

`listener, e := websocket.Listen("tls", args...)`  

Type websocket.Listener is inherited from net.Listener.Any incoming connection can be accepted by Accept method,just like net.Listener:  

`conn, e := listener.Accept()`  

Type websocket.Conn is inherited from net.Conn.It has several useful methods to transport data.

#### Read

Read is the main function to consume stream data from the other side.You can call it just like net.Conn.It will call the underlying Read method,and process websocket frames.The signature of this method:  

`func (conn *Conn) Read(p []byte) (d []byte, op byte, ok bool, err error)`

Read method will never return the application data inside ping-pong frames.

#### Close

Close closes the connection to the other side.You may find it useless in server-side context.  

#### SendText

SendText sends a text frame to the client-side.All the frame sending methods return an error,if any.  

#### SendBinary

SendBinary sends a binary frame to the client-side.  

#### SendPing

SendPing sends a ping-frame to the client-side.You can get more information of ping-pong frames in rfc6455.  

#### SendPong

SendPong sends a pong-frame to the client-side.  

#### Connections

Connections is an arry of alive connections.It save all the connections' pointer.You can get and modify it in your own program(Not recommanded, though).  

#### Broadcast

Broadcast receive a string or a byte slice.It will judge the specific type of the argument,and then,call the SendText or SendBinary method on every connection in the Connections.It is useful at most time.If a connection in the Connections is not alive any more,it will be closed and aborted.  

### Client-side

You can create a websocket-client in this way:  

`client, err := websokcet.Connect(uri)`

It return a Client object and an error.Uri is in this format:  

`ws://host:port`  

Attention:The client-side websocket **do not** support tls connection.The method of client-side websocket is similar to server-side.  

#### Read

Read calls the underlying method of connect.  

#### Close

Signature:

`func (client *Client) Close(closeCode code, reason string)`

Close closes the client and send a closeframe to server.Check the rfc6455 for more information.  

#### SendText

#### SendBinary

#### SendPing

#### SendPong

## LICENSE

MIT
