package parser

import (
	"bytes"
	"fmt"
	"net"
	"strconv"
	"strings"
)

type Response struct {
	conn *net.Conn
	buff    []byte
	buffLen int
	start   int
	end     int
	PData
}

//实例化
func NewResponse(conn *net.Conn, buffLen int) *Response {
	return &Response{
		conn: conn,
		buffLen: buffLen,
		buff: make([]byte, buffLen),
		PData: PData{
			Line: make(map[string]string),
			Header: make(map[string]string),
		},
	}
}

//解析响应行
func (r *Response) parseLine() (isOK bool, err error) {
	index := bytes.Index(r.buff, []byte{byte('\r'), byte('\n')})
	if index == -1 {
		//没有解析到\r\n返回继续读取
		return
	}

	responseLine := string(r.buff[:index])
	arr := strings.Split(responseLine, " ")
	if len(arr) < 3 {
		return false, fmt.Errorf("bad response line:%v", responseLine)
	}
	r.Line["version"] = arr[0]
	r.Line["status"] = arr[1]
	r.Line["description"] = strings.Join(arr[2:], "")
	r.start += index + 2
	return true, nil
}

//解析响应头
func (r *Response) parseHeader() bool {
	if r.start == r.end {
		return false
	}
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

//解析响应体
func (r *Response) parseBody() (isOk bool, err error) {
	//判断请求头中是否指明了请求体的数据长度
	contentLenStr := r.Header["Content-Length"]
	contentLen, err := strconv.ParseInt(contentLenStr, 10, 64)
	if err != nil {
		return false, fmt.Errorf("parse Content-Length error:%s", contentLenStr)
	}
	if contentLen > int64(r.end - r.start) {
		//请求体长度不够，返回继续读取
		return false, nil
	}
	r.Body = string(r.buff[r.start:int64(r.start)+contentLen])
	r.start += int(contentLen)
	return true, nil
}

//解析分块传输的数据
func (r *Response)parseChunked() bool {
	for {
		index := bytes.Index(r.buff[r.start:], []byte{byte('\r'), byte('\n')})
		if index == -1 {
			return false
		}
		bLen, _ := strconv.ParseUint(string(r.buff[r.start:r.start+index]), 16, 32)
		blockLen := int(bLen)

		if blockLen == 0 {
			r.start += index + 4
			//最后一个块数据
			return true
		}else{
			//判断是否读取了完整的块数据
			if blockLen > r.end - r.start {
				return false
			}
			r.start += index + 2 + blockLen + 2
		}
	}
}

//读取http响应
func (r *Response) Read(accept chan PData) (err error) {
	for  {
		if r.end == r.buffLen {
			//缓冲区的容量存不了一条请求的数据
			return fmt.Errorf("response is too large:%v", r)
		}
		rLen, err := (*r.conn).Read(r.buff[r.end:])
		if err != nil {
			//连接关闭了
			return nil
		}
		r.end += rLen

		LOOP:
		//解析响应行
		isOk, err := r.parseLine()
		if err != nil {
			return fmt.Errorf("parse response line error:%v", err)
		}
		if !isOk {
			continue
		}

		//解析响应头
		isOk = r.parseHeader()
		if !isOk {
			r.reset()
			continue
		}

		//如果有CONTENT-LENGTH头部，解析响应体
		if _, isOk := r.Header["Content-Length"]; isOk {
			isOk, err := r.parseBody()
			if err != nil {
				return fmt.Errorf("parse response body error:%v", err)
			}
			//读取http请求体未成功
			if !isOk {
				r.reset()
				continue
			}
		}else{
			//如果有TRANSFER-ENCODING头部，且值为chunked，解析响应体
			if tEncoding, isOk := r.Header["Transfer-Encoding"]; isOk && tEncoding == "chunked" {
				//解析分块传输的数据
				readOk := r.parseChunked()
				if !readOk {
					r.reset()
					continue
				}
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
func (r *Response) reset() {
	r.start = 0
	r.Line = make(map[string]string)
	r.Header = make(map[string]string)
	r.Body = ""
	r.BinaryData = ""
}
