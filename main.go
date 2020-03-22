package main

import (
	"fmt"
	"github.com/matchseller/http-proxy-server/server"
	"github.com/matchseller/http-proxy-server/util"
	"os"
	"strconv"
	"strings"
	"sync"
)

func main(){
	args, err := parseArgs()
	if err != nil {
		panic(err)
	}
	p := server.NewServer(args["cAddr"].(string), args["pAddr"].(string), args["cCount"].(int), args["pCount"].(int))
	var wg sync.WaitGroup
	wg.Add(2)
	go p.RunAcceptor(&wg)
	go p.RunProxy(&wg)
	fmt.Println("Proxy server is running!")
	wg.Wait()
	fmt.Println("Proxy stopped unexpectedly!")
}

func parseArgs() (map[string]interface{}, error) {
	args := make(map[string]interface{})
	argKeys := []string{"cAddr", "pAddr", "cCount", "pCount"}
	necessaryKeys := argKeys[:2]
	if len(os.Args) < 3 {
		return nil, fmt.Errorf("not enough parameters:%v",os.Args[1:])
	}

	for i := 1; i < len(os.Args); i++ {
		argArr := strings.Split(os.Args[i], "=")
		if len(argArr) != 2 {
			return nil, fmt.Errorf("incorrect parameter format:%v", os.Args[i])
		}
		if !util.StringInArray(argKeys, argArr[0]) {
			return nil, fmt.Errorf("incorrect parameter key:%v", argArr[0])
		}
		if strings.TrimSpace(argArr[1]) == "" {
			return args, fmt.Errorf("the key '%v''s value cannot be empty", argArr[0])
		}

		args[argArr[0]] = argArr[1]
	}

	for _, v := range necessaryKeys {
		if _, isOk := args[v]; !isOk {
			return nil, fmt.Errorf("missing parameter:%v", v)
		}
	}

	if val, isOk := args[argKeys[2]]; isOk {
		cCount, err := strconv.ParseInt(val.(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("argument must be an integer:%v", val)
		}
		args[argKeys[2]] = int(cCount)
	}else{
		args[argKeys[2]] = 1000
	}

	if val, isOk := args[argKeys[3]]; isOk {
		pCount, err := strconv.ParseInt(val.(string), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("argument must be an integer:%v", val)
		}
		args[argKeys[3]] = int(pCount)
	}else{
		args[argKeys[3]] = 1000
	}

	return args, nil
}
