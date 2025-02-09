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
	chatId := flag.Int64("c", -1, "chat id")
	msg := flag.String("m", "", "message body")
	u := flag.String("u", "", "url")
	replyIdVal := flag.Int("r", -1, "reply id")
	
	flag.Parse()
	
	var replyId *int
	if *replyIdVal != -1 {
		replyId = replyIdVal
	}

	ur, err := url.Parse(*u)
	if err != nil {
		fmt.Println("error on parsing url")
		os.Exit(1)
	}
	ur = ur.JoinPath("/sendMsg")

	req := commands.SendMsgReq{
		Msg:     *msg,
		ChatId:  *chatId,
		ReplyId: replyId,
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
