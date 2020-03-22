package response

import (
	"fmt"
	"net/http"
)

func (r *Response)Error() {
	r.header["Server"] = "Proxy"
	r.header["Content-Type"] = "text/html"

	description := fmt.Sprintf("%d %s", r.status, http.StatusText(r.status))
	r.body = fmt.Sprintf(`<html>
		<head><title>%s</title></head>
		<body>
		<center><h1>%s</h1></center>
		<hr><center>Proxy</center>
		</body>
		</html>`, description, description)
	r.Send()
}
