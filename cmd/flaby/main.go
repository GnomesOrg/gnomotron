package main

import (
	"bytes"
	"encoding/json"
	"flabergnomebot/commands"
	"flag"
	"fmt"
	"io"
	"net/http"
)

func main() {
    chatId := flag.Int64("c", -1, "chat id")
    msg := flag.String("m", "", "message body")
    url := flag.String("u", "", "url")

    flag.Parse()

    *url = *url + "/sendMsg"

    req := commands.SendMsgReq{
    	Msg:    *msg,
    	ChatId: *chatId,
    }

    data, err := json.Marshal(req)
    if err != nil {
        panic(err)
    }

    resp, err := http.Post(*url, "application/json", bytes.NewReader(data))
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
