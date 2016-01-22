package pool

import (
	"fmt"
	"net"
	"time"
)

//wrappedConn modify the behavior of net.Conn's Write() method and Close() method
//while other methods can be accessed transparently.
type wrappedConn struct {
	net.Conn
	pool       *blockingPool
	unusable   bool
	closed     bool
	lastAccess time.Time
	liveTime   time.Duration
	//net.Conn generator
	factory Factory
}

// genChild
func (c *wrappedConn) genChild() *wrappedConn {
	return &wrappedConn{
		nil,
		c.pool,
		true,
		true,
		time.Now(),
		c.liveTime,
		c.factory,
	}
}

//TODO
func (c *wrappedConn) Close() error {
	if c.closed {
		return fmt.Errorf("close conn fail: conn is already closed")
	}

	if c.unusable {
		c.destory()
		c.pool.putBottom(c)

	} else {
		c.pool.putTop(c)
	}

	c.closed = true
	return nil
}

// getInactiveNetConn
func (c *wrappedConn) checkIdle() bool {
	if time.Since(c.lastAccess) > c.liveTime && c.Conn != nil {
		return true
	}
	return false
}

// activate
func (c *wrappedConn) activate() error {
	c.closed = false
	if c.Conn == nil {
		if conn, err := c.factory(); err != nil {
			c.Close()
			return err
		} else {
			c.Conn = conn
			c.unusable = false
			c.lastAccess = time.Now()
		}
	}
	return nil
}

// destory
func (c *wrappedConn) destory() {
	conn := c.Conn
	c.Conn = nil
	c.unusable = true
	if conn != nil {
		conn.Close()
	}
}

//Write checkout the error returned from the origin Write() method.
//If the error is not nil, the connection is marked as unusable.
func (c *wrappedConn) Write(b []byte) (n int, err error) {
	if c.closed {
		err = fmt.Errorf("write conn fail: conn is already closed")
		return
	}
	//c.Conn is certainly not nil
	n, err = c.Conn.Write(b)
	if err != nil {
		c.unusable = true
	} else {
		c.lastAccess = time.Now()
	}
	return
}

//Read works the same as Write.
func (c *wrappedConn) Read(b []byte) (n int, err error) {
	if c.closed {
		err = fmt.Errorf("read conn fail: conn is already closed")
		return
	}

	//c.Conn is certainly not nil
	n, err = c.Conn.Read(b)
	if err != nil {
		c.unusable = true
	} else {
		c.lastAccess = time.Now()
	}

	return
}

//wrap wraps net.Conn and start a delayClose goroutine
func (p *blockingPool) wrap(conn net.Conn, livetime time.Duration, factory Factory) (c *wrappedConn) {
	c = &wrappedConn{
		conn,
		p,
		true,
		true,
		time.Now(),
		livetime,
		factory,
	}
	if c.Conn != nil {
		c.unusable = false
	}
	return
}
