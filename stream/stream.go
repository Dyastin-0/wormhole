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

// StreamHTTPWithInspect is StreamWithContext with HTTP request-response inspection.
func StreamHTTPWithInspect(
	ctx context.Context,
	src, dst net.Conn,
	onRequest func(start time.Time, method, path string, status int),
) error {
	defer src.Close()
	defer dst.Close()

	type reqEntry struct {
		req   *http.Request
		start time.Time
	}

	reqCh := make(chan reqEntry)
	respCh := make(chan *http.Response)
	errCh := make(chan error, 2)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case entry := <-reqCh:
				resp := <-respCh
				onRequest(entry.start, entry.req.Method, entry.req.URL.Path, resp.StatusCode)
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

			reqCh <- reqEntry{req: req, start: reqStart}

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
		if err == io.EOF {
			return nil
		}
		return err
	}
}
