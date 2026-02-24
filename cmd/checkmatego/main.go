package main

import "checkmatego/internal/uci"

func main() {
	handler := uci.NewHandler()
	handler.Run()
}
