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
	var once sync.Once
	var err error

	setError := func(e error) {
		once.Do(func() {
			err = e
			cancel()
		})
	}

	// Copy src -> dst
	wg.Add(1)
	go func() {
		defer wg.Done()
		if copyErr := copyWithContext(ctx, dst, src); copyErr != nil {
			setError(copyErr)
		}
	}()

	// Copy dst -> src
	wg.Add(1)
	go func() {
		defer wg.Done()
		if copyErr := copyWithContext(ctx, src, dst); copyErr != nil {
			setError(copyErr)
		}
	}()

	wg.Wait()

	closeConnection(src)
	closeConnection(dst)

	return err
}

// StreamWithContext is Stream with context.
func StreamWithContext(ctx context.Context, src, dst io.ReadWriter) error {
	var wg sync.WaitGroup
	var once sync.Once
	var err error

	localCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	setError := func(e error) {
		once.Do(func() {
			err = e
			cancel()
		})
	}

	// Copy src -> dst
	wg.Add(1)
	go func() {
		defer wg.Done()
		if copyErr := copyWithContext(localCtx, dst, src); copyErr != nil {
			setError(copyErr)
		}
	}()

	// Copy dst -> src
	wg.Add(1)
	go func() {
		defer wg.Done()
		if copyErr := copyWithContext(localCtx, src, dst); copyErr != nil {
			setError(copyErr)
		}
	}()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		setError(ctx.Err())
		cancel()
		<-done
	}

	closeConnection(src)
	closeConnection(dst)

	return err
}

// copyWithContext performs io.Copy with context cancellation.
func copyWithContext(ctx context.Context, dst io.Writer, src io.Reader) error {
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if conn, ok := src.(net.Conn); ok {
			conn.SetReadDeadline(time.Now().Add(1 * time.Second))
		}

		n, readErr := src.Read(buf)
		if n > 0 {
			if conn, ok := dst.(net.Conn); ok {
				conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
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

			if readErr == io.EOF {
				return nil
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
