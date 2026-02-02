// Package stream implements a bidirectional stream.
package stream

import (
	"bufio"
	"context"
	"io"
	"net"
	"net/http"
	"time"
)

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

// Request embeds http.Request along with a request start timestamp.
type Request struct {
	*http.Request
	start time.Time
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
	// this way we can do a bidirectional copy similar to
	// Stream and StreamWithContext by sniffing the requests and responses
	// and replaying it on the conns.
	reqCh := make(chan *Request, 100)
	respCh := make(chan *http.Response, 100)
	closeCh := make(chan struct{})
	errCh := make(chan error, 2)

	// This matches request with its coresponding response.
	go func() {
		for {
			select {
			case <-closeCh:
			case <-ctx.Done():
				return
			case entry := <-reqCh:
				resp := <-respCh
				onRequest(entry.start, entry.Method, entry.URL.Path, resp.StatusCode)
			}
		}
	}()

	go func() {
		srcTee, srcReader := NewTeeConn(src)
		br := bufio.NewReader(srcReader)

		for {
			reqStart := time.Now()
			req, err := http.ReadRequest(br)
			if err != nil {
				errCh <- err
				return
			}
			req.Body.Close()
			br.Discard(br.Buffered())

			reqCh <- &Request{Request: req, start: reqStart}

			if _, err := io.Copy(dst, srcTee); err != nil {
				errCh <- err
				return
			}
		}
	}()

	go func() {
		dstTee, dstReader := NewTeeConn(dst)
		br := bufio.NewReader(dstReader)

		for {
			resp, err := http.ReadResponse(br, nil)
			if err != nil {
				errCh <- err
				return
			}

			resp.Body.Close()
			br.Discard(br.Buffered())

			respCh <- resp

			if _, err := io.Copy(src, dstTee); err != nil {
				errCh <- err
				return
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
