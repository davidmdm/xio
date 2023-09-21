package xio

type copyoptions struct {
	WaitForLastOp bool
	bufferSize    int
}

type CopyOption func(*copyoptions)

func WaitForLastOp(value bool) CopyOption {
	return func(c *copyoptions) {
		c.WaitForLastOp = value
	}
}

func BufferSize(value int) CopyOption {
	return func(c *copyoptions) {
		c.bufferSize = value
	}
}
