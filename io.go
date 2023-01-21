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
// is the number of bytes written at the time of cancelation unless the option WaitForLastWrite is present.
// The goroutine will exit at the end of the current read/write cycle, however itis possible that a write is
// still in effect and that the total number of bytes written will be greater than reported.
func Copy(ctx context.Context, dst io.Writer, src io.Reader, opts ...CopyOption) (n int, err error) {
	options := copyoptions{
		WaitForLastWrite: false,
		bufferSize:       4096,
	}
	for _, apply := range opts {
		apply(&options)
	}

	var atomicN atomic.Int64
	errCh := make(chan error, 1)

	if options.WaitForLastWrite {
		defer func() {
			if endErr := <-errCh; endErr != nil {
				err = endErr
			}
			n = int(atomicN.Load())
		}()
	}

	go func() {
		defer close(errCh)

		buf := make([]byte, options.bufferSize)
		for {
			rn, rErr := src.Read(buf)
			if rn > 0 {
				wn, wErr := dst.Write(buf[:rn])
				if wn < 0 || wn > rn {
					errCh <- errInvalidWrite
					return
				}

				atomicN.Add(int64(wn))

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
		return int(atomicN.Load()), ctx.Err()
	case err := <-errCh:
		return int(atomicN.Load()), err
	}
}

func ReadAll(ctx context.Context, src io.Reader) ([]byte, error) {
	var dst bytes.Buffer
	_, err := Copy(ctx, &dst, src, WaitForLastWrite(true))
	return dst.Bytes(), err
}
