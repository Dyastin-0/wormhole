// Package proxy implements a bidirectional stream.
package proxy

import (
	"bufio"
	"context"
	"io"
	"net/http"
	"sync"
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

// StreamWithContextInspect is StreamWithContext with HTTP response inspection.
// onStatus is called once when HTTP response status is detected (from dst->src direction).
func StreamWithContextInspect(ctx context.Context, src, dst io.ReadWriter, onStatus func(int)) error {
	errch := make(chan error, 2)
	once := sync.Once{}

	// Copy src -> dst
	go func() {
		_, err := io.Copy(dst, src)
		errch <- err
		closeConnection(dst)
		closeConnection(src)
	}()

	// Copy dst -> src with inspection
	go func() {
		br := bufio.NewReader(dst)
		resp, err := http.ReadResponse(br, nil)
		if err == nil {
			once.Do(func() {
				onStatus(resp.StatusCode)
			})
		}

		// Copy remaining data from buffered reader
		_, err = io.Copy(src, br)
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
