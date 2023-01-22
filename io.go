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
// given to it is canceled. If the context is canceled, Copy will wait for the current read/write cycle to end
// then exit unless explicitly passed the option "WaitForLastWrite(false)". If WaitForLastWrite is false, Copy
// will exit as soon as the context is canceled and the value of n will reflect the number of byte written to dst
// at the time of the cancelation and but is not guaranteed to be the total bytes written to dst by the time to
// write goroutine exits. Use WaitForLastWrite(false) if src or dst is slow and you do not care about the total
// amount of bytes written to dst if a cancelation occurs.
func Copy(ctx context.Context, dst io.Writer, src io.Reader, opts ...CopyOption) (n int64, err error) {
	options := copyoptions{
		WaitForLastWrite: true,
		bufferSize:       32 * 1024, // same as io/io.go
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
			n = atomicN.Load()
		}()
	}

	if lr, ok := src.(*io.LimitedReader); ok && int64(options.bufferSize) > lr.N {
		if lr.N < 1 {
			options.bufferSize = 1
		} else {
			options.bufferSize = int(lr.N)
		}
	}

	buf := make([]byte, options.bufferSize)

	go func() {
		defer close(errCh)
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
		return atomicN.Load(), ctx.Err()
	case err := <-errCh:
		return atomicN.Load(), err
	}
}

// CopyN behaves like io.CopyN but is cancelable via a context. The same options as Copy can be passed to CopyN.
func CopyN(ctx context.Context, dst io.Writer, src io.Reader, n int64, opts ...CopyOption) (written int64, err error) {
	written, err = Copy(ctx, dst, io.LimitReader(src, n), opts...)
	if written == n {
		return n, nil
	}
	if written < n && err == nil {
		// src stopped early; must have been EOF.
		err = io.EOF
	}
	return
}

// ReadAll works like io.Readall but is cancelable via a context.
func ReadAll(ctx context.Context, src io.Reader) ([]byte, error) {
	var dst bytes.Buffer
	_, err := Copy(ctx, &dst, src, WaitForLastWrite(true))
	return dst.Bytes(), err
}
