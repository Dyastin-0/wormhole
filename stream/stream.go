// Package stream implements a bidirectional stream and parsing utilities.
package stream

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	maxInspectSize  = 1 * 1024 * 1024
	requestIDHeader = "X-Wormhole-Request-ID"
)

// HTTPEvent is emitted by StreamHTTPWithContext for each completed request/response pair.
type HTTPEvent struct {
	ID          string
	Start       time.Time
	Duration    uint64
	Method      string
	Path        string
	Status      int
	RespSize    int64
	ReqHeaders  http.Header
	ReqBody     []byte
	RespHeaders http.Header
	RespBody    []byte
}

// pendingReq holds the server-side state for an in-flight request, keyed by
// position in the FIFO pipeline. HTTP/1.1 responses are always returned in
// the same order as requests, so a channel acts as an ordered queue.
type pendingReq struct {
	id         string
	start      time.Time
	method     string
	path       string
	reqHeaders http.Header
	reqBody    []byte
}

// Request wraps http.Request with a start time.
type Request struct {
	*http.Request
	start time.Time
}

// Response wraps http.Request with a size.
type Response struct {
	*http.Response
	Size int64
}

// CountWriter count bytes as it writes.
type CountWriter struct {
	w     io.Writer
	count int64
}

func (cw *CountWriter) Write(p []byte) (int, error) {
	n, err := cw.w.Write(p)
	cw.count += int64(n)
	return n, err
}

type LimitedTeeReader struct {
	R io.Reader
	W io.Writer
	N int64
}

func NewHTTPEventWithoutcontext(method, path string, status int) HTTPEvent {
	return HTTPEvent{
		ID:     uuid.NewString(),
		Start:  time.Now(),
		Method: method,
		Path:   path,
		Status: status,
	}
}

// Stream handles bidirectional streaming between src and dst.
func Stream(src, dst net.Conn) error {
	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(dst, src)
		errc <- err
		dst.Close()
	}()

	_, err := io.Copy(src, dst)
	errc <- err
	src.Close()

	err2 := <-errc

	if err != nil && err != io.EOF {
		return err
	}
	if err2 != nil && err2 != io.EOF {
		return err2
	}
	return nil
}

// StreamWithContext is Stream with context cancellation support.
func StreamWithContext(ctx context.Context, src, dst net.Conn) error {
	errc := make(chan error, 2)

	go func() {
		_, err := io.Copy(dst, src)
		errc <- err
	}()

	go func() {
		_, err := io.Copy(src, dst)
		errc <- err
	}()

	select {
	case <-ctx.Done():
		src.Close()
		dst.Close()
		return ctx.Err()

	case err := <-errc:
		src.Close()
		dst.Close()

		<-errc

		if err == io.EOF {
			return nil
		}
		return err
	}
}

func (l *LimitedTeeReader) Read(p []byte) (n int, err error) {
	n, err = l.R.Read(p)
	if n > 0 && l.N > 0 {
		toCopy := min(int64(n), l.N)
		l.W.Write(p[:toCopy])
		l.N -= toCopy
	}
	return n, err
}

// StreamHTTPWithContext proxies HTTP/1.1 traffic between src (downstream client)
// and dst (upstream backend), injects a unique request ID header, captures
// headers and bodies for inspection, and emits a correlated HTTPEvent per
// request/response pair. eventch may be nil if inspection is not needed.
func StreamHTTPWithContext(
	ctx context.Context,
	src, dst net.Conn,
	eventch chan<- any,
) error {
	brSrc := bufio.NewReader(src)
	brDst := bufio.NewReader(dst)

	bcSrc := &BuffConn{Conn: src, r: brSrc}
	bcDst := &BuffConn{Conn: dst, r: brDst}

	defer src.Close()
	defer dst.Close()

	// pendingCh acts as the FIFO queue between the request and response goroutines.
	pendingCh := make(chan pendingReq, 16)
	errCh := make(chan error, 2)
	upgradeCh := make(chan struct{})

	go func() {
		defer close(pendingCh)
		for {
			req, err := http.ReadRequest(brSrc)
			if err != nil {
				errCh <- err
				return
			}

			id := uuid.NewString()
			req.Header.Set(requestIDHeader, id)

			isUpgrade := req.Header.Get("Upgrade") == "websocket"
			start := time.Now()

			var reqBodyBuf bytes.Buffer
			req.Body = io.NopCloser(&LimitedTeeReader{
				R: req.Body,
				W: &reqBodyBuf,
				N: maxInspectSize,
			})

			reqHeaders := req.Header.Clone()

			if err := req.Write(dst); err != nil {
				errCh <- err
				return
			}

			select {
			case pendingCh <- pendingReq{
				id:         id,
				start:      start,
				method:     req.Method,
				path:       req.URL.Path,
				reqHeaders: reqHeaders,
				reqBody:    reqBodyBuf.Bytes(),
			}:
			default:
				// Drop if consumer is behind; we never block the proxy path.
			}

			if isUpgrade {
				return
			}
		}
	}()

	go func() {
		cw := &CountWriter{w: src}
		for {
			resp, err := http.ReadResponse(brDst, nil)
			if err != nil {
				errCh <- err
				return
			}

			isUpgrade := resp.StatusCode == http.StatusSwitchingProtocols

			var respBodyBuf bytes.Buffer
			resp.Body = io.NopCloser(&LimitedTeeReader{
				R: resp.Body,
				W: &respBodyBuf,
				N: maxInspectSize,
			})

			respHeaders := resp.Header.Clone()
			status := resp.StatusCode

			cw.count = 0
			if err := resp.Write(cw); err != nil {
				errCh <- err
				return
			}

			respSize := cw.count

			if eventch != nil {
				pending, ok := <-pendingCh
				if ok {
					select {
					case eventch <- &HTTPEvent{
						ID:          pending.id,
						Start:       pending.start,
						Method:      pending.method,
						Path:        pending.path,
						Status:      status,
						RespSize:    respSize,
						ReqHeaders:  pending.reqHeaders,
						ReqBody:     pending.reqBody,
						RespHeaders: respHeaders,
						RespBody:    respBodyBuf.Bytes(),
					}:
					default:
					}
				}
			}

			if isUpgrade {
				close(upgradeCh)
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-upgradeCh:
		return StreamWithContext(ctx, bcSrc, bcDst)
	case err := <-errCh:
		if err == io.EOF {
			return nil
		}
		return err
	}
}
