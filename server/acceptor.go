package server

import (
	"github.com/matchseller/http-proxy-server/log"
	"github.com/matchseller/http-proxy-server/parser"
	"github.com/matchseller/http-proxy-server/response"
	"github.com/matchseller/http-proxy-server/util"
	"net"
	"net/http"
	"sync"
)

type message struct {
	fd int
	binaryData string
}

type client struct {
	fd int
	rData chan parser.PData
	buffLen int
	pServer *ProxyServer
}

func (p *ProxyServer)RunAcceptor(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp4", p.cAddr)
	if err != nil {
		log.MyLogger.Panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.MyLogger.Println("accept error:", err)
			continue
		}

		fd, err := p.client.AddConn(&conn)
		if err != nil {
			response.NewResponse(http.StatusServiceUnavailable, &conn).Error()
			conn.Close()
			log.MyLogger.Println(err)
			continue
		}
		c := client{
			fd: fd,
			rData: make(chan parser.PData),
			buffLen:10*1024*1024,
			pServer: p,
		}
		go c.handle()
	}
}

func (c *client)handle() {
	defer func() {
		c.pServer.client.CloseConn(c.fd)
		close(c.rData)
	}()
	go c.acceptRequest()
	err := parser.NewRequest(c.pServer.client.GetConn(c.fd), c.buffLen).Read(c.rData)
	if err != nil {
		log.MyLogger.Println("request parse error：", err)
		response.NewResponse(http.StatusBadRequest, c.pServer.client.GetConn(c.fd)).Error()
	}
}

func (c *client)acceptRequest() {
	for {
		for i := 0; i < 10; i++ {
			data, isOk := <- c.rData
			if !isOk {
				return
			}

			host, isOk := data.Header["Host"]
			if !isOk {
				response.NewResponse(http.StatusBadRequest, c.pServer.client.GetConn(c.fd)).Error()
				continue
			}

			conn := c.pServer.proxyClient.GetConn(util.Md5String(host))
			if conn == nil {
				response.NewResponse(http.StatusNotFound, c.pServer.client.GetConn(c.fd)).Error()
				continue
			}

			//将请求数据和源连接发送给消息通道
			c.pServer.msgChan <- message{
				fd:c.fd,
				binaryData:data.BinaryData,
			}
		}
	}
}
