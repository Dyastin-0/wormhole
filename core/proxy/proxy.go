// Package proxy implements a bidirectional stream
package proxy

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"
)

var bufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

// Stream handles bidirectional stream between src and dst.
func Stream(src, dst io.ReadWriter) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	errch := make(chan error, 1)

	// Copy src -> dst
	go func() {
		err := CopyWithContext(ctx, dst, src)
		select {
		case errch <- err:
		default:
		}
	}()
	// Copy dst -> src
	go func() {
		err := CopyWithContext(ctx, src, dst)
		select {
		case errch <- err:
		default:
		}
	}()

	err := <-errch
	closeConnection(src)
	closeConnection(dst)

	return err
}

// StreamWithContext is Stream with context.
func StreamWithContext(ctx context.Context, src, dst io.ReadWriter) error {
	errch := make(chan error, 1)
	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Copy src -> dst
	go func() {
		err := CopyWithContext(localCtx, dst, src)
		select {
		case errch <- err:
		default:
		}
	}()
	// Copy dst -> src
	go func() {
		err := CopyWithContext(localCtx, src, dst)
		select {
		case errch <- err:
		default:
		}
	}()

	go func() {
		err := CopyWithContext(ctx, src, dst)
		select {
		case errch <- err:
		default:
		}
	}()

	err := <-errch
	closeConnection(src)
	closeConnection(dst)

	return err
}

// CopyWithContext performs io.Copy with context cancellation.
func CopyWithContext(ctx context.Context, dst, src io.ReadWriter) error {
	bufPtr := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(bufPtr)
	buf := *bufPtr

	done := make(chan struct{})
	defer close(done)

	if conn, ok := src.(net.Conn); ok {
		go func() {
			select {
			case <-ctx.Done():
				conn.Close()
			case <-done:
				return
			}
		}()
	}

	// Exponential backoff
	backoff := 10 * time.Millisecond
	maxBackoff := 1 * time.Second
	maxRetries := 5
	consecutiveTimeouts := 0

	if conn, ok := src.(net.Conn); ok {
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetReadDeadline(deadline)
		} else {
			conn.SetReadDeadline(time.Now().Add(5 * time.Second))
		}
	}

	if conn, ok := dst.(net.Conn); ok {
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetWriteDeadline(deadline)
		} else {
			conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			consecutiveTimeouts = 0
			backoff = 10 * time.Millisecond

			_, writeErr := dst.Write(buf[:n])
			if writeErr != nil {
				return writeErr
			}
		}

		if readErr != nil {
			if netErr, ok := readErr.(net.Error); ok && netErr.Timeout() {
				consecutiveTimeouts++

				if consecutiveTimeouts >= maxRetries {
					return fmt.Errorf("max timeout retries exceeded: %w", readErr)
				}

				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
					backoff *= 2
					if backoff > maxBackoff {
						backoff = maxBackoff
					}
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
