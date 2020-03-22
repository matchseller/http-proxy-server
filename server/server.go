package server

import (
	"fmt"
	"github.com/matchseller/http-proxy-server/util"
	"net"
	"sync"
)

type clientConnManager struct {
	connections sync.Map
	connCount int
	maxConnCount int
	maxHandleCount int
	rw sync.RWMutex
	maxFd int
	increment int
}

type proxyClientConnManager struct {
	connections sync.Map
	connCount int
	maxConnCount int
	rw sync.RWMutex
}

type ProxyServer struct {
	cAddr string
	pAddr string
	msgChan chan message
	client *clientConnManager
	proxyClient *proxyClientConnManager
}

type connection struct {
	conn *net.Conn
	count int
}

func NewServer(cAddr, pAddr string, maxClientCount, maxProxyClientCount int) *ProxyServer {
	return &ProxyServer{
		cAddr:cAddr,
		pAddr:pAddr,
		msgChan:make(chan message),
		client:&clientConnManager{
			maxConnCount: maxClientCount,
			maxHandleCount:5,
			maxFd:        16000000,
		},
		proxyClient:&proxyClientConnManager{
			maxConnCount: maxProxyClientCount,
		},
	}
}

func (c *clientConnManager)AddConn(conn *net.Conn) (fd int, err error) {
	if c.connCount == c.maxConnCount {
		return 0, fmt.Errorf("exceeded the maximum number of connections")
	}
	c.increment++
	if c.increment > c.maxFd {
		c.increment = 1
	}
	c.connections.Store(c.increment, &connection{
		conn:  conn,
		count: 0,
	})

	fd = c.increment
	c.rw.Lock()
	c.connCount++
	c.rw.Unlock()
	return fd, nil
}

func (c *clientConnManager)handleCountIncrement(fd int) {
	val, isOk := c.connections.Load(fd)
	if !isOk {
		return
	}
	conn := val.(*connection)
	conn.count++
	if conn.count == c.maxHandleCount {
		//关闭该连接
		c.CloseConn(fd)
	}else {
		c.connections.Store(fd, conn)
	}
}

func (c *clientConnManager)CloseConn(fd int) {
	val, isOk := c.connections.Load(fd)
	if isOk {
		conn := val.(*connection)
		(*conn.conn).Close()
		c.connections.Delete(fd)

		c.rw.Lock()
		c.connCount--
		c.rw.Unlock()
	}
}

func (c *clientConnManager)GetConn(fd int) *net.Conn {
	val, isOk := c.connections.Load(fd)
	if !isOk {
		return nil
	}
	return val.(*connection).conn
}

func (p *proxyClientConnManager)AddConn(host string, conn *net.Conn) (string, error) {
	if p.connCount == p.maxConnCount {
		return "", fmt.Errorf("exceeded the maximum number of connections")
	}
	fd := util.Md5String(host)
	p.connections.Store(fd, conn)
	p.rw.Lock()
	p.connCount++
	p.rw.Unlock()
	return fd, nil
}

func (p *proxyClientConnManager)CloseConn(fd string) {
	val, isOk := p.connections.Load(fd)
	if isOk {
		conn := val.(*net.Conn)
		(*conn).Close()
		p.connections.Delete(fd)

		p.rw.Lock()
		p.connCount--
		p.rw.Unlock()
	}
}

func (p *proxyClientConnManager)GetConn(fd string) *net.Conn {
	val, isOk := p.connections.Load(fd)
	if !isOk {
		return nil
	}
	return val.(*net.Conn)
}
