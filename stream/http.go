package stream

import (
	"bufio"
	"net"
	"net/http"
)

type HTTPConn struct {
	*TeeConn
	Request *http.Request
}

func HTTP(conn net.Conn) (httpConn *HTTPConn, err error) {
	c, rd := NewTeeConn(conn)

	httpConn = &HTTPConn{TeeConn: c}
	if httpConn.Request, err = http.ReadRequest(bufio.NewReader(rd)); err != nil {
		return httpConn, err
	}

	httpConn.Request.Body.Close()

	return httpConn, err
}
