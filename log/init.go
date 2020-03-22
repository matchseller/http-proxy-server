package log

import (
	"github.com/matchseller/http-proxy-server/util"
	"log"
	"os"
)

var MyLogger *log.Logger

func init() {
	if MyLogger == nil {
		util.CheckWorkDir()
		path := os.Getenv("PROXY_SERVER_WORK_DIR") + "/log/"
		//创建系统日志目录
		if !util.Exists(path) {
			//创建目录
			err := os.MkdirAll(path, os.ModePerm)
			if err != nil {
				panic(err)
			}
		}
		file := path + "system.txt"
		logFile, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0766)
		if nil != err {
			panic(err)
		}
		MyLogger = log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
	}
}
