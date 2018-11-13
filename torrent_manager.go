package main

import (
	"log"
	"strings"

	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/spf13/viper"
)

type TorrentInfo struct {
	Name      string `json:"name"`
	Completed int64  `json:"completed"`
	Total     int64  `json:"total"`
	InfoHash  string `json:"info_hash"`
	VideoFile *torrent.File
}

type TorrentManager struct {
	client *torrent.Client
}

func (t *TorrentManager) Start() {
	clientConfig := torrent.NewDefaultClientConfig()
	clientConfig.DataDir = viper.GetString("download_dir")
	t.client, _ = torrent.NewClient(clientConfig)
}

func (t *TorrentManager) Stop() {
	t.client.Close()
}

func (t *TorrentManager) Download(url string) (T *TorrentInfo, err error) {
	var newTorrent *torrent.Torrent
	var magnet string

	log.Print("Starting to download: ", url)

	if strings.HasPrefix(url, "magnet") {

		log.Print("It is a magnet link! 🤘")

		magnet = url
	} else {
		log.Print("It is a URL, trying to download the torrent file 😴")
		filePath, err := DownloadFile(url)

		if err != nil {
			log.Print("There was an error downloading the torrent file 😔")
			return nil, err
		}

		log.Print("Torrent file downloaded! Converting to magnet...")

		metaInfo, err := metainfo.LoadFromFile(filePath)

		if err != nil {
			log.Print("Error reading metainfo from torrent: ", err)
			return nil, err
		}

		info, err := metaInfo.UnmarshalInfo()

		if err != nil {
			log.Print("Error unmarshalling info: ", err)
			return nil, err
		}

		magnet = metaInfo.Magnet(info.Name, metaInfo.HashInfoBytes()).String()

		log.Print("Magnet for the torrent: ", magnet)
	}

	newTorrent, err = t.client.AddMagnet(magnet)

	if err != nil {
		log.Print("There was an error adding the torrent 😔")
		return nil, err
	}

	log.Print("Getting torrent information...")

	<-newTorrent.GotInfo()
	newTorrent.DownloadAll()

	var videoFile *torrent.File

	for _, file := range newTorrent.Files() {
		if strings.HasSuffix(file.Path(), "mp4") {
			videoFile = file
		}
	}

	videoFile.SetPriority(torrent.PiecePriorityNow)

	log.Print("Torrent added successfully! 🤘 -> ", newTorrent.Info().Name)

	torrentInfo := &TorrentInfo{
		Name:      newTorrent.Info().Name,
		Completed: newTorrent.BytesCompleted(),
		Total:     newTorrent.Length(),
		InfoHash:  newTorrent.InfoHash().String(),
		VideoFile: videoFile,
	}

	return torrentInfo, nil
}

func (t *TorrentManager) GetTorrents() (r []TorrentInfo) {
	torrents := make([]TorrentInfo, 0)

	for _, torrent := range t.client.Torrents() {
		torrents = append(torrents, TorrentInfo{
			Name:      torrent.Info().Name,
			Completed: torrent.BytesCompleted(),
			Total:     torrent.Length(),
			InfoHash:  torrent.InfoHash().String(),
		})
	}

	return torrents
}

func (t *TorrentManager) GetTorrentFromHash(infoHash string) (f *torrent.Torrent) {
	for _, torrent := range t.client.Torrents() {
		if torrent.InfoHash().String() == infoHash {
			return torrent
		}
	}

	log.Print("No torrent found with this infoHash 😔")
	return nil
}

func (t *TorrentManager) FindVideoFileFromHash(infoHash string) (f *torrent.File) {
	for _, torrent := range t.client.Torrents() {
		if torrent.InfoHash().String() == infoHash {

		}
	}

	log.Print("No video file found with this infoHash 😔")
	return nil
}
