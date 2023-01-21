package xio

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestCopy(t *testing.T) {
	t.Run("copy with context cancelation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		unblockWrite := make(chan struct{})

		var writeTotal int

		n, err := Copy(
			ctx,
			WriterFunc(func(b []byte) (int, error) {
				cancel()
				<-unblockWrite
				writeTotal += len(b)
				return len(b), nil
			}),
			ReaderFunc(func(b []byte) (int, error) {
				return len(b), nil
			}),
		)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected error to be context canceled but got %v", err)
		}

		if n != 0 {
			t.Fatalf("expected n to be 0 but got %d", n)
		}

		close(unblockWrite)

		time.Sleep(time.Millisecond)

		if writeTotal != 4096 {
			t.Fatalf("expected write total to be 4096 but got %d", writeTotal)
		}
	})

	t.Run("Copy with wait for last write", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		unblockWrite := make(chan struct{})

		time.AfterFunc(20*time.Millisecond, func() { close(unblockWrite) })

		n, err := Copy(
			ctx,
			WriterFunc(func(b []byte) (int, error) {
				cancel()
				<-unblockWrite
				return len(b), nil
			}),
			ReaderFunc(func(b []byte) (int, error) {
				return len(b), nil
			}),
			WaitForLastWrite(true),
		)

		if !errors.Is(err, context.Canceled) {
			t.Fatalf("expected error to be context canceled but got %v", err)
		}

		if n != 4096 {
			t.Fatalf("expected n to be 4096 but got %d", n)
		}
	})
}

type ReaderFunc func([]byte) (int, error)

func (fn ReaderFunc) Read(data []byte) (int, error) { return fn(data) }

type WriterFunc func([]byte) (int, error)

func (fn WriterFunc) Write(data []byte) (int, error) { return fn(data) }
