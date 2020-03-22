package response

import (
	"fmt"
	"github.com/matchseller/http-proxy-server/log"
	"net"
	"net/http"
	"strconv"
)

type Response struct {
	status int
	httpVersion string
	buff []byte
	buffLen int
	end int
	conn *net.Conn
	header map[string]string
	body string
}

func NewResponse(status int, conn *net.Conn) *Response {
	buffLen := 10*1024*1024
	return &Response{
		status:  status,
		buff:    make([]byte, buffLen),
		buffLen: buffLen,
		conn:    conn,
		header:  make(map[string]string),
		httpVersion: "HTTP/1.1",
	}
}

func (r *Response)SetVersion(version string) *Response {
	r.httpVersion = version
	return r
}

func (r *Response)SetHeader(header map[string]string) *Response {
	r.header = header
	return r
}

func (r *Response)SetBody(body string) *Response {
	r.body = body
	return r
}

func (r *Response)GetBuff() []byte {
	r.build()
	return r.buff[:r.end]
}

func (r *Response) Send() {
	r.build()
	_, err := (*r.conn).Write(r.buff[:r.end])
	if err != nil {
		log.MyLogger.Println("response error:", err)
	}
}

//发送响应
func (r *Response) build() {
	r.buildLine()
	r.buildHeader()

	copy(r.buff[r.end:], r.body)
	r.end += len(r.body)
}

//构造响应行
func (r *Response) buildLine() {
	line := fmt.Sprintf("%s %d %s\r\n", r.httpVersion, r.status, http.StatusText(r.status))
	copy(r.buff[r.end:], line)
	r.end += len(line)
}

//构造响应头
func (r *Response) buildHeader() {
	contentLen := int64(len(r.body))
	if contentLen > 0 {
		r.header["Content-Length"] = strconv.FormatInt(contentLen, 10)
	}
	for k, v := range r.header {
		str := fmt.Sprintf("%s: %v\r\n", k, v)
		copy(r.buff[r.end:], str)
		r.end += len(str)
	}
	r.end += 2
}
