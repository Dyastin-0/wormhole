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

// Request wraps http.Request with a start time.
type Request struct {
	*http.Request
	start time.Time
}

// StreamHTTPWithInspect is StreamWithContext with HTTP request-response inspection.
func StreamHTTPWithInspect(
	ctx context.Context,
	src, dst net.Conn,
	onRequest func(start time.Time, method, path string, status int, length int64),
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
			case entry := <-reqCh:
				resp := <-respCh
				onRequest(entry.start, entry.Method, entry.URL.Path, resp.StatusCode, resp.ContentLength)
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
