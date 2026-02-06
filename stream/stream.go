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
)

const maxInspectSize = 5 * 1024 * 1024

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

// StreamHTTPWithContext is StreamWithContext that inspects HTTP traffic and
// sends them to the internal channels and calls the callback function.
func StreamHTTPWithContext(
	ctx context.Context,
	src, dst net.Conn,
	onRequest func(start time.Time, method, path string, status int),
) error {
	brSrc := bufio.NewReader(src)
	brDst := bufio.NewReader(dst)

	bcSrc := &BuffConn{Conn: src, r: brSrc}
	bcDst := &BuffConn{Conn: dst, r: brDst}

	defer src.Close()
	defer dst.Close()

	reqCh := make(chan *Request, 1)
	respCh := make(chan *http.Response, 1)
	errCh := make(chan error, 2)

	stopHTTP := make(chan struct{})

	go func() {
		defer close(reqCh)
		for {
			req, err := http.ReadRequest(brSrc)
			if err != nil {
				errCh <- err
				return
			}

			isUpgrade := req.Header.Get("Upgrade") == "websocket"
			reqStart := time.Now()

			if err := req.Write(dst); err != nil {
				errCh <- err
				return
			}

			select {
			case reqCh <- &Request{Request: req, start: reqStart}:
			default:
			}

			if isUpgrade {
				return
			}
		}
	}()

	go func() {
		defer close(respCh)
		for {
			resp, err := http.ReadResponse(brDst, nil)
			if err != nil {
				errCh <- err
				return
			}

			if err := resp.Write(src); err != nil {
				errCh <- err
				return
			}

			select {
			case respCh <- resp:
			default:
			}

			if resp.StatusCode == http.StatusSwitchingProtocols {
				close(stopHTTP)
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errCh:
			if err == io.EOF {
				return nil
			}
			return err
		case req, ok := <-reqCh:
			if !ok {
				goto RawStream
			}

			resp, ok := <-respCh
			if !ok {
				goto RawStream
			}

			onRequest(req.start, req.Method, req.URL.Path, resp.StatusCode)

			if resp.StatusCode == http.StatusSwitchingProtocols {
				goto RawStream
			}
		case <-stopHTTP:
			goto RawStream
		}
	}

RawStream:
	return StreamWithContext(ctx, bcSrc, bcDst)
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

// StreamHTTPWithRequestResponseContext is StreamWithHTTPContext that inspects HTTP traffic
// and sends them to an external channels without blocking.
func StreamHTTPWithRequestResponseContext(
	ctx context.Context,
	src, dst net.Conn,
	responsech chan any,
	requestch chan *http.Request,
) error {
	brSrc := bufio.NewReader(src)
	brDst := bufio.NewReader(dst)

	bcSrc := &BuffConn{Conn: src, r: brSrc}
	bcDst := &BuffConn{Conn: dst, r: brDst}

	defer src.Close()
	defer dst.Close()

	errCh := make(chan error, 2)
	upgradeCh := make(chan struct{})

	go func() {
		for {
			req, err := http.ReadRequest(brSrc)
			if err != nil {
				errCh <- err
				return
			}

			isUpgrade := req.Header.Get("Upgrade") == "websocket"

			var reqBodyBuf bytes.Buffer
			req.Body = io.NopCloser(&LimitedTeeReader{
				R: req.Body,
				W: &reqBodyBuf,
				N: maxInspectSize,
			})

			if err := req.Write(dst); err != nil {
				errCh <- err
				return
			}

			tuiReq := req.Clone(ctx)
			tuiReq.Body = io.NopCloser(bytes.NewReader(reqBodyBuf.Bytes()))
			select {
			case requestch <- tuiReq:
			default:
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

			cw.count = 0
			var respBodyBuf bytes.Buffer
			resp.Body = io.NopCloser(&LimitedTeeReader{
				R: resp.Body,
				W: &respBodyBuf,
				N: maxInspectSize,
			})

			if err := resp.Write(cw); err != nil {
				errCh <- err
				return
			}

			tuiResp := *resp
			tuiResp.Body = io.NopCloser(bytes.NewReader(respBodyBuf.Bytes()))
			tuiResp.Header = resp.Header.Clone()
			select {
			case responsech <- &Response{Response: &tuiResp, Size: cw.count}:
			default:
			}

			if resp.StatusCode == http.StatusSwitchingProtocols {
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
