package main

import (
	"bytes"
	"encoding/json"
	"flabergnomebot/commands"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
)

func main() {
	msg := flag.String("m", "", "message body")
	u := flag.String("u", "", "url")

	flag.Parse()

	ur, err := url.Parse(*u)
	if err != nil {
		fmt.Println("error on parsing url")
		os.Exit(1)
	}
	ur = ur.JoinPath("/sendBcMsg")

	req := commands.SendBcMsgReq{
		Msg: *msg,
	}

	data, err := json.Marshal(req)
	if err != nil {
		panic(err)
	}

	resp, err := http.Post(ur.String(), "application/json", bytes.NewReader(data))
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	fmt.Println("Response status:", resp.Status)
	fmt.Println("Response body:", string(b))
}
