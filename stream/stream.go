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
	requestIDHeader = "X-Wormhole-Request-Id"
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
type pendingRequest struct {
	id         string
	start      time.Time
	method     string
	path       string
	reqHeaders http.Header
	reqBody    []byte
}

type responseMeta struct {
	id      string
	status  int
	size    int64
	headers http.Header
	body    []byte
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

func (l *LimitedTeeReader) Read(p []byte) (n int, err error) {
	n, err = l.R.Read(p)
	if n > 0 && l.N > 0 {
		toCopy := min(int64(n), l.N)
		l.W.Write(p[:toCopy])
		l.N -= toCopy
	}
	return n, err
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

// StreamHTTPWithContext proxies HTTP/1.1 traffic between src (downstream client)
// and dst (upstream backend), injects a unique request ID header, captures
// headers and bodies for inspection, and emits a correlated HTTPEvent per
// request/response pair.
func StreamHTTPWithContext(
	ctx context.Context,
	src, dst net.Conn,
	eventch chan<- any,
	server bool,
) error {
	brSrc := bufio.NewReader(src)
	brDst := bufio.NewReader(dst)
	bwSrc := bufio.NewWriter(src)
	bwDst := bufio.NewWriter(dst)

	defer src.Close()
	defer dst.Close()

	reqMetaCh := make(chan pendingRequest, 1024)
	respMetaCh := make(chan responseMeta, 1024)
	errCh := make(chan error, 2)
	upgradeCh := make(chan struct{})

	go func() {
		for {
			select {
			case req := <-reqMetaCh:
				resp := <-respMetaCh

				if eventch != nil {
					eventID := resp.id
					if eventID == "" {
						eventID = req.id
					}

					eventch <- &HTTPEvent{
						ID:          eventID,
						Start:       req.start,
						Method:      req.method,
						Path:        req.path,
						Status:      resp.status,
						RespSize:    resp.size,
						ReqHeaders:  req.reqHeaders,
						ReqBody:     req.reqBody,
						RespHeaders: resp.headers,
						RespBody:    resp.body,
					}
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer close(reqMetaCh)
		for {
			req, err := http.ReadRequest(brSrc)
			if err != nil {
				errCh <- err
				return
			}

			meta := pendingRequest{
				id:         req.Header.Get(requestIDHeader),
				start:      time.Now(),
				method:     req.Method,
				path:       req.URL.Path,
				reqHeaders: req.Header.Clone(),
			}

			var buf bytes.Buffer
			req.Body = io.NopCloser(&LimitedTeeReader{
				R: req.Body,
				W: &buf,
				N: maxInspectSize,
			})

			if err := req.Write(bwDst); err != nil {
				errCh <- err
				return
			}
			bwDst.Flush()
			meta.reqBody = buf.Bytes()

			reqMetaCh <- meta

			if req.Header.Get("Upgrade") == "websocket" {
				return
			}
		}
	}()

	go func() {
		cw := &CountWriter{w: bwSrc}
		for {
			resp, err := http.ReadResponse(brDst, nil)
			if err != nil {
				errCh <- err
				return
			}

			if !server {
				resp.Header.Set(requestIDHeader, uuid.NewString())
			}

			var buf bytes.Buffer
			resp.Body = io.NopCloser(&LimitedTeeReader{
				R: resp.Body,
				W: &buf,
				N: maxInspectSize,
			})

			headers := resp.Header.Clone()
			status := resp.StatusCode
			id := resp.Header.Get(requestIDHeader)

			cw.count = 0
			if err := resp.Write(cw); err != nil {
				errCh <- err
				return
			}
			bwSrc.Flush()

			select {
			case respMetaCh <- responseMeta{
				id:      id,
				status:  status,
				size:    cw.count,
				headers: headers,
				body:    buf.Bytes(),
			}:
			default:
			}

			if status == http.StatusSwitchingProtocols {
				close(upgradeCh)
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-upgradeCh:
		return StreamWithContext(ctx, &BuffConn{src, brSrc}, &BuffConn{dst, brDst})
	case err := <-errCh:
		if err == io.EOF {
			return nil
		}
		return err
	}
}
