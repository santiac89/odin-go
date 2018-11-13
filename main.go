package main

import (
	"fmt"

	"github.com/spf13/viper"
)

var torrentManager TorrentManager

func readConfig() {
	viper.AddConfigPath(".")
	viper.SetConfigName("config")
	err := viper.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %s \n", err))
	}
}

func main() {
	readConfig()

	torrentManager = TorrentManager{}
	torrentManager.Start()
	defer torrentManager.Stop()

	apiChannel := make(chan string)
	go StartWebServer(apiChannel)
	<-apiChannel
}
