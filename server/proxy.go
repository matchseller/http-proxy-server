package server

import (
	"bytes"
	"github.com/matchseller/http-proxy-server/log"
	"github.com/matchseller/http-proxy-server/parser"
	"github.com/matchseller/http-proxy-server/response"
	"github.com/matchseller/http-proxy-server/util"
	"net"
	"net/http"
	"strings"
	"sync"
)

const pwd = "e10adc3949ba59abbe56e057f20f883e"	//123456

type validator struct {
	buffLen int
	buff []byte
	start int
	end int
}

type proxyClient struct {
	host string
	fd string		//与代理客户端的连接标识
	deadChan chan bool	//通知与代理客户端连接断开的消息
	responseOk chan bool
	clientFd int		//与当前请求客户端的连接标识
	validator validator
	pServer *ProxyServer
}

func (p *ProxyServer)RunProxy(wg *sync.WaitGroup) {
	defer wg.Done()
	listener, err := net.Listen("tcp4", p.pAddr)
	if err != nil {
		log.MyLogger.Panic(err)
	}
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.MyLogger.Println("accept error:", err)
			continue
		}

		pManager := &proxyClient{
			deadChan: make(chan bool),
			responseOk:make(chan bool),
			validator:validator{
				buffLen: 256,
				buff:    make([]byte, 256),
				start:   0,
				end:     0,
			},
			pServer:p,
		}

		go pManager.handle(&conn)
	}
}

func (p *proxyClient)handle(conn *net.Conn) {
	var err error
	p.validator.end, err = (*conn).Read(p.validator.buff)
	if err != nil {
		(*conn).Close()
		return
	}
	host, msg := p.validate(conn)
	if msg != "" {
		log.MyLogger.Println("authentication failed:", msg)
		(*conn).Write([]byte("400 " + msg))
		(*conn).Close()
		return
	}
	_, err = (*conn).Write([]byte("200 ok"))
	if err != nil {
		(*conn).Close()
		return
	}
	p.host = host
	go p.listen()
	go p.readResponse()
}

//验证客户端代理的身份，保存代理的服务域名
func (p *proxyClient)validate(conn *net.Conn) (host, msg string) {
	var err error

	if p.validator.end == p.validator.buffLen {
		return "", "authentication package is too long"
	}
	//按规则解析数据包：第一行为代理的服务域名，第二行为密码
	index := bytes.Index(p.validator.buff, []byte{byte('\r'), byte('\n')})
	if index == -1 {
		return "", "authentication package format is incorrect"
	}
	host = string(p.validator.buff[p.validator.start:p.validator.start+index])
	p.validator.start += index + 2
	if host == "" {
		return "", "authentication package format is incorrect"
	}
	index = bytes.Index(p.validator.buff[p.validator.start:], []byte{byte('\r'), byte('\n')})
	if index == -1 {
		return "", "authentication package format is incorrect"
	}
	password := p.validator.buff[p.validator.start:p.validator.start+index]
	if string(password) != pwd {
		return "", "incorrect password"
	}

	c := p.pServer.proxyClient.GetConn(util.Md5String(host))
	if c != nil {
		//已经有代理客户端代理该域名了
		return "", "duplicate proxy"
	}

	p.fd, err = p.pServer.proxyClient.AddConn(host, conn)
	if err != nil {
		//代理客户端数量超过限定值
		return "", "server overload"
	}
	return
}

func (p *proxyClient)listen() {
	for {
		select {
		case <-p.deadChan:
			p.pServer.proxyClient.CloseConn(p.fd)
			return
		case msg := <-p.pServer.msgChan:
			conn := p.pServer.proxyClient.GetConn(p.fd)
			_, err := (*conn).Write([]byte(msg.binaryData))
			if err != nil {
				response.NewResponse(http.StatusServiceUnavailable, p.pServer.client.GetConn(msg.fd))
				continue
			}
			p.clientFd = msg.fd
			<- p.responseOk
		}
	}
}

func (p *proxyClient)readResponse() {
	rData := make(chan parser.PData)
	go p.handleResponse(rData)
	err := parser.NewResponse(p.pServer.proxyClient.GetConn(p.fd), 10*1024*1024).Read(rData)
	if err != nil {
		log.MyLogger.Println("parse response error:", err)
		response.NewResponse(http.StatusInternalServerError, p.pServer.client.GetConn(p.clientFd))
	}
	p.deadChan <- true
	close(rData)
}

func (p *proxyClient)handleResponse(rData chan parser.PData)  {
	for {
		data, isOk := <- rData
		if !isOk {
			break
		}
		conn := p.pServer.client.GetConn(p.clientFd)
		if conn != nil {
			(*conn).Write([]byte(data.BinaryData))
		}
		if val, isOk := data.Header["Connection"]; (isOk && val == "close") || strings.ToUpper(data.Line["version"]) == "HTTP/1.0" {
			//非持久连接响应后关闭该连接
			p.pServer.client.CloseConn(p.clientFd)
		}else {
			//持久连接响应后检查在该连接上处理的事务次数是否超过上限，超过就关闭连接
			p.pServer.client.handleCountIncrement(p.clientFd)
		}
		p.responseOk <- true
	}
}
