package parser

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Request struct {
	conn *net.Conn
	buff    []byte
	buffLen int
	start   int
	end     int
	PData
}

type PData struct {
	Line map[string]string	//请求行
	Header map[string]string	//请求头
	Body string	//请求体
	BinaryData string
}

//实例化
func NewRequest(conn *net.Conn, buffLen int) *Request {
	return &Request{
		conn: conn,
		PData: PData{
			Line:   make(map[string]string),
			Header: make(map[string]string),
		},
		buffLen: buffLen,
		buff: make([]byte, buffLen),
	}
}

//解析请求行
func (r *Request) parseLine() (isOK bool, err error) {
	index := bytes.Index(r.buff, []byte{byte('\r'), byte('\n')})
	if index == -1 {
		//没有解析到\r\n返回继续读取
		return
	}
	//解析请求行
	requestLine := string(r.buff[:index])
	arr := strings.Split(requestLine, " ")
	if len(arr) != 3 {
		return false, fmt.Errorf("bad request line")
	}
	r.Line["method"] = arr[0]
	r.Line["url"] = arr[1]
	r.Line["version"] = arr[2]

	r.start += index + 2
	return true, nil
}

//解析请求头
func (r *Request) parseHeader() bool {
	index := bytes.Index(r.buff[r.start:], []byte{byte('\r'), byte('\n'), byte('\r'), byte('\n')})
	if index == -1 {
		return false
	}
	headerStr := string(r.buff[r.start:r.start+index])
	requestHeader := strings.Split(headerStr, "\r\n")
	for _, v := range requestHeader {
		arr := strings.Split(v, ":")
		if len(arr) < 2 {
			continue
		}
		r.Header[arr[0]] = strings.Trim(strings.Join(arr[1:], ":"), " ")
	}
	r.start += index + 4
	return true
}

//解析请求体
func (r *Request) parseBody() (isOk bool, err error) {
	contentLength := r.Header["Content-Length"]
	cLength, err := strconv.ParseInt(contentLength, 10, 64)
	if err != nil {
		return false, fmt.Errorf("parse Content-Length error:%s", contentLength)
	}
	cLen := int(cLength)
	if cLen > r.end - r.start {
		//请求体未读取完整，返回继续读取
		return false, nil
	}
	r.Body = string(r.buff[r.start:r.start+cLen])
	r.start += cLen
	return true, nil
}

//读取http请求
func (r *Request) Read(accept chan PData) (err error) {
	for  {
		if r.end == r.buffLen {
			//缓冲区的容量存不了一条请求的数据
			return fmt.Errorf("request is too large:%v", r)
		}
		rLen, err := (*r.conn).Read(r.buff[r.end:])
		if err != nil {
			//连接关闭了
			return nil
		}
		r.end += rLen

	LOOP:
		//解析请求行
		isOk, err := r.parseLine()
		if err != nil {
			return fmt.Errorf("parse request line error:%v", err)
		}
		if !isOk {
			continue
		}

		//解析请求头
		isOk = r.parseHeader()
		if !isOk {
			r.reset()
			continue
		}

		//如果有CONTENT-LENGTH头部，解析请求体
		if _, isOk := r.Header["Content-Length"]; isOk {
			isOk, err := r.parseBody()
			if err != nil {
				return fmt.Errorf("parse request body error:%v", err)
			}
			//读取http请求体未成功
			if !isOk {
				r.reset()
				continue
			}
		}
		r.BinaryData = string(r.buff[:r.start])
		accept <- r.PData
		if r.start != r.end {
			goto LOOP
		}
		r.end = 0
		r.reset()
	}
}

//重置
func (r *Request) reset() {
	r.start = 0
	r.Line = make(map[string]string)
	r.Header = make(map[string]string)
	r.Body = ""
	r.BinaryData = ""
}
