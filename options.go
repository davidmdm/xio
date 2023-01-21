package xio

type copyoptions struct {
	WaitForLastWrite bool
	bufferSize       int
}

type CopyOption func(*copyoptions)

func WaitForLastWrite(value bool) CopyOption {
	return func(c *copyoptions) {
		c.WaitForLastWrite = value
	}
}

func BufferSize(value int) CopyOption {
	return func(c *copyoptions) {
		c.bufferSize = value
	}
}
