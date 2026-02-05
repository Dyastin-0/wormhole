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

const maxInspectSize = 20 * 1024 * 1024

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

// StreamHTTPWithInspect is StreamWithContext with HTTP request-response inspection.
func StreamHTTPWithInspect(
	ctx context.Context,
	src, dst net.Conn,
	onRequest func(start time.Time, method, path string, status int),
) error {
	defer src.Close()
	defer dst.Close()

	// channels for coordinating request/response cycles,
	// this way we can do a "bidirectional copy" similar to
	// Stream and StreamWithContext.
	reqCh := make(chan *Request, 16)
	respCh := make(chan *http.Response, 16)
	closeCh := make(chan struct{})
	errCh := make(chan error, 2)

	// This matches request with its coresponding response,
	// becuase HTTP/1.1 is inherently sequential, meaning a client
	// must wait for a request's response before sending another request.
	go func() {
		for {
			select {
			case <-closeCh:
				return
			case <-ctx.Done():
				return
			case req := <-reqCh:
				resp := <-respCh
				onRequest(req.start, req.Method, req.URL.Path, resp.StatusCode)
			}
		}
	}()

	go func() {
		br := bufio.NewReader(src)

		for {
			req, err := http.ReadRequest(br)
			if err != nil {
				errCh <- err
				return
			}

			reqStart := time.Now()

			err = req.Write(dst)
			if err != nil {
				errCh <- err
				return
			}

			select {
			case reqCh <- &Request{Request: req, start: reqStart}:
			default:
			}
		}
	}()

	go func() {
		br := bufio.NewReader(dst)

		for {
			resp, err := http.ReadResponse(br, nil)
			if err != nil {
				errCh <- err
				return
			}

			err = resp.Write(src)
			if err != nil {
				errCh <- err
				return
			}

			select {
			case respCh <- resp:
			default:
			}
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		close(closeCh)
		if err == io.EOF {
			return nil
		}
		return err
	}
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

func StreamHTTPWithRequestResponseInspect(
	ctx context.Context,
	src, dst net.Conn,
	responsech chan any,
	requestch chan *http.Request,
) error {
	defer src.Close()
	defer dst.Close()

	errCh := make(chan error, 2)

	go func() {
		br := bufio.NewReader(src)
		for {
			req, err := http.ReadRequest(br)
			if err != nil {
				errCh <- err
				return
			}

			var reqBodyBuf bytes.Buffer
			req.Body = io.NopCloser(&LimitedTeeReader{
				R: req.Body,
				W: &reqBodyBuf,
				N: maxInspectSize,
			})

			err = req.Write(dst)
			if err != nil {
				errCh <- err
				return
			}

			tuiReq := req.Clone(ctx)
			tuiReq.Body = io.NopCloser(bytes.NewReader(reqBodyBuf.Bytes()))

			select {
			case requestch <- tuiReq:
			default:
			}
		}
	}()

	go func() {
		br := bufio.NewReader(dst)
		cw := &CountWriter{w: src}

		for {
			resp, err := http.ReadResponse(br, nil)
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

			err = resp.Write(cw)
			if err != nil {
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
		}
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errCh:
		if err == io.EOF {
			return nil
		}
		return err
	}
}
