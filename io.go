package xio

import (
	"bytes"
	"context"
	"errors"
	"io"
	"sync/atomic"
)

// errInvalidWrite means that a write returned an impossible count.
var errInvalidWrite = errors.New("invalid write result")

// Copy attempts to copy all of src into dst. It uses a goroutine to do so, and will exit early if the context
// given to it is canceled. When the error is due to a context cancelation or timeout the number of bytes written
// is the number of bytes written at the time of cancelation. The goroutine will exit at the end of the current read/write cycle, however it
// is possible that a write is still in effect and that the total number of bytes written will be greater than reported.
func Copy(ctx context.Context, dst io.Writer, src io.Reader) (int, error) {
	var n atomic.Int64
	errCh := make(chan error, 1)

	go func() {
		defer close(errCh)

		buf := make([]byte, 4096)
		for {
			rn, rErr := src.Read(buf)
			if rn > 0 {
				wn, wErr := dst.Write(buf[:rn])
				if wn < 0 || wn > rn {
					errCh <- errInvalidWrite
					return
				}

				n.Add(int64(wn))

				if wErr != nil {
					errCh <- wErr
					return
				}
			}

			if rErr != nil {
				if rErr != io.EOF {
					errCh <- rErr
				}
				return
			}
			if err := ctx.Err(); err != nil {
				errCh <- err
				return
			}
		}
	}()

	select {
	case <-ctx.Done():
		return int(n.Load()), ctx.Err()
	case err := <-errCh:
		return int(n.Load()), err
	}
}

func ReadAll(ctx context.Context, r io.Reader) ([]byte, error) {
	var w bytes.Buffer
	_, err := Copy(ctx, &w, r)
	return w.Bytes(), err
}
