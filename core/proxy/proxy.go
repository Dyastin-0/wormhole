// Package proxy implements a bidirectional stream.
package proxy

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"time"
)

// Stream handles bidirectional streaming between src and dst.
func Stream(src, dst io.ReadWriteCloser) error {
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
func StreamWithContext(ctx context.Context, src, dst io.ReadWriteCloser) error {
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
	src, dst io.ReadWriteCloser,
	onRequest func(start time.Time, method, path string, status int),
) error {
	srcBr := bufio.NewReader(src)
	dstBr := bufio.NewReader(dst)

	for {
		select {
		case <-ctx.Done():
			src.Close()
			dst.Close()
			return ctx.Err()
		default:
		}

		start := time.Now()

		req, err := http.ReadRequest(srcBr)
		if err != nil {
			src.Close()
			dst.Close()
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				return nil
			}
			return err
		}

		method, path := req.Method, req.URL.Path

		if err = req.Write(dst); err != nil {
			src.Close()
			dst.Close()
			return err
		}

		resp, err := http.ReadResponse(dstBr, req)
		if err != nil {
			src.Close()
			dst.Close()
			return err
		}

		status := resp.StatusCode

		if err := resp.Write(src); err != nil {
			resp.Body.Close()
			src.Close()
			dst.Close()
			return err
		}

		resp.Body.Close()
		onRequest(start, method, path, status)
	}
}
