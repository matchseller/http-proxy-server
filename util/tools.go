package util

import (
	"crypto/md5"
	"encoding/hex"
	"os"
)

//判断环境变量是否存在
func CheckWorkDir() {
	if os.Getenv("PROXY_SERVER_WORK_DIR") == "" {
		panic("请先设置PROXY_SERVER_WORK_DIR环境变量")
	}
}

// 判断所给路径文件/文件夹是否存在
func Exists(path string) bool {
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		return false
	}
	return true
}

//获取md5值
func Md5String(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func StringInArray(arr []string, str string) bool {
	for _, v := range arr {
		if v == str {
			return true
		}
	}
	return false
}
