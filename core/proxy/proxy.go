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
func Stream(src, dst io.ReadWriter) error {
	errch := make(chan error, 2)
	// Copy src -> dst
	go func() {
		_, err := io.Copy(dst, src)
		errch <- err
	}()
	// Copy dst -> src
	go func() {
		_, err := io.Copy(src, dst)
		errch <- err
	}()
	err := <-errch
	closeConnection(src)
	closeConnection(dst)
	<-errch
	return err
}

// StreamWithContext is Stream with context cancellation support.
func StreamWithContext(ctx context.Context, src, dst io.ReadWriter) error {
	errch := make(chan error, 2)
	// Copy src -> dst
	go func() {
		_, err := io.Copy(dst, src)
		errch <- err
		closeConnection(dst)
		closeConnection(src)
	}()
	// Copy dst -> src
	go func() {
		_, err := io.Copy(src, dst)
		errch <- err
		closeConnection(src)
		closeConnection(dst)
	}()
	select {
	case <-ctx.Done():
		closeConnection(src)
		closeConnection(dst)
		<-errch
		<-errch
		return ctx.Err()
	case err := <-errch:
		err2 := <-errch
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			return err
		}
		return err2
	}
}

// StreamHTTPWithInspect is StreamWithContext with HTTP request-response inspection.
func StreamHTTPWithInspect(ctx context.Context, src, dst io.ReadWriter, onRequest func(start time.Time, method, path string, status int)) error {
	var srcBr *bufio.Reader

	if br, ok := src.(interface{ GetReader() *bufio.Reader }); ok {
		srcBr = br.GetReader()
	} else {
		srcBr = bufio.NewReader(src)
	}

	dstBr := bufio.NewReader(dst)

	errch := make(chan error, 1)

	go func() {
		defer func() {
			closeConnection(src)
			closeConnection(dst)
		}()

		for {
			select {
			case <-ctx.Done():
				errch <- ctx.Err()
				return
			default:
			}

			start := time.Now()

			req, err := http.ReadRequest(srcBr)
			if err != nil {
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					errch <- nil
					return
				}
				errch <- err
				return
			}

			method, path := req.Method, req.URL.Path

			if err = req.Write(dst); err != nil {
				errch <- err
				return
			}

			resp, err := http.ReadResponse(dstBr, req)
			if err != nil {
				errch <- err
				return
			}

			status := resp.StatusCode

			if err := resp.Write(src); err != nil {
				resp.Body.Close()
				errch <- err
				return
			}

			resp.Body.Close()

			onRequest(start, method, path, status)

			if !shouldKeepAlive(req, resp) {
				errch <- nil
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		closeConnection(src)
		closeConnection(dst)
		return ctx.Err()
	case err := <-errch:
		closeConnection(src)
		closeConnection(dst)
		return err
	}
}

func shouldKeepAlive(req *http.Request, resp *http.Response) bool {
	if req.Close || resp.Close {
		return false
	}
	if resp.Header.Get("Connection") == "close" {
		return false
	}
	return true
}

// closeConnection safely closes a connection if it implements io.Closer.
func closeConnection(conn io.ReadWriter) {
	if cr, ok := conn.(interface{ CloseRead() error }); ok {
		cr.CloseRead()
	}
	if cw, ok := conn.(interface{ CloseWrite() error }); ok {
		cw.CloseWrite()
	}
	if closer, ok := conn.(io.Closer); ok {
		closer.Close()
	}
}
