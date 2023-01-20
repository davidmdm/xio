package xio

type copyoptions struct {
	WaitForLastWrite bool
}

type CopyOption func(*copyoptions)

func WaitForLastWrite(value bool) CopyOption {
	return func(c *copyoptions) {
		c.WaitForLastWrite = value
	}
}
