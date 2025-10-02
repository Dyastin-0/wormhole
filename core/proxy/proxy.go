// Package proxy implements a bidirectional stream
package proxy

import (
	"context"
	"io"
	"net"
	"sync"
	"time"
)

// Stream handles bidirectional stream between src and dst.
func Stream(src, dst io.ReadWriter) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	errch := make(chan error, 1)

	// Copy src -> dst
	wg.Go(func() {
		errch <- CopyWithContext(ctx, dst, src)
	})

	// Copy dst -> src
	wg.Go(func() {
		errch <- CopyWithContext(ctx, src, dst)
	})

	wg.Wait()

	closeConnection(src)
	closeConnection(dst)

	return <-errch
}

// StreamWithContext is Stream with context.
func StreamWithContext(ctx context.Context, src, dst io.ReadWriter) error {
	var wg sync.WaitGroup
	errch := make(chan error, 1)

	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Copy src -> dst
	wg.Go(func() {
		errch <- CopyWithContext(localCtx, dst, src)
	})

	// Copy dst -> src
	wg.Go(func() {
		errch <- CopyWithContext(localCtx, src, dst)
	})

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		errch <- ctx.Err()
		cancel()
		<-done
	}

	closeConnection(src)
	closeConnection(dst)

	return <-errch
}

// CopyWithContext performs io.Copy with context cancellation.
func CopyWithContext(ctx context.Context, dst, src io.ReadWriter) error {
	buf := make([]byte, 32*1024)

	if conn, ok := src.(net.Conn); ok {
		go func() {
			<-ctx.Done()
			conn.Close()
		}()
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if conn, ok := src.(net.Conn); ok {
			if deadline, ok := ctx.Deadline(); ok {
				conn.SetReadDeadline(deadline)
			} else {
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			}
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			if conn, ok := dst.(net.Conn); ok {
				if deadline, ok := ctx.Deadline(); ok {
					conn.SetWriteDeadline(deadline)
				} else {
					conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
				}
			}
			_, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
		}

		if readErr != nil {
			if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
				select {
				case <-ctx.Done():
					return ctx.Err()
				default:
					continue
				}
			}

			return readErr
		}
	}
}

// closeConnection safely closes a connection if it implements io.Closer.
func closeConnection(conn io.ReadWriter) {
	if closer, ok := conn.(io.Closer); ok {
		closer.Close()
	}
}
