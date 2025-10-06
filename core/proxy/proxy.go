// Package proxy implements a bidirectional stream.
package proxy

import (
	"context"
	"io"
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
	done := make(chan struct{})

	go func() {
		select {
		case <-ctx.Done():
			closeConnection(src)
			closeConnection(dst)
		case <-done:
		}
	}()

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

	close(done)

	closeConnection(src)
	closeConnection(dst)

	<-errch

	if ctx.Err() != nil {
		return ctx.Err()
	}

	return err
}

// closeConnection safely closes a connection if it implements io.Closer.
func closeConnection(conn io.ReadWriter) {
	if closer, ok := conn.(io.Closer); ok {
		closer.Close()
	}
}
