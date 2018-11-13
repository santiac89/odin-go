package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/ricksancho/srt2vtt"
	"github.com/spf13/viper"
)

type WatchResponse struct {
	HTML string `json:"html"`
	Path string `json:"path"`
}

func torrents(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(torrentManager.GetTorrents())
}

func subtitles(w http.ResponseWriter, r *http.Request) {
	filePath := r.URL.Query().Get("path")

	file, err := os.Open(filePath)

	if err == nil {
		reader, err := srt2vtt.NewReader(file)

		if err == nil {
			reader.WriteTo(w)
		} else {
			json.NewEncoder(w).Encode(err)
		}
	} else {
		json.NewEncoder(w).Encode(err)
	}
}

// func saveFile(w http.ResponseWriter, file multipart.File, handle *multipart.FileHeader) {
// 	data, err := ioutil.ReadAll(file)
// 	if err != nil {
// 		fmt.Fprintf(w, "%v", err)
// 		return
// 	}

// 	err = ioutil.WriteFile("./files/"+handle.Filename, data, 0666)
// 	if err != nil {
// 		fmt.Fprintf(w, "%v", err)
// 		return
// 	}
// 	// jsonResponse(w, http.StatusCreated, "File uploaded successfully!.")
// }

func uploadSubtitles(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")

	file, handle, err := r.FormFile("file")

	if err != nil {
		fmt.Fprintf(w, "%v", err)
		return
	}

	defer file.Close()

	if filepath.Ext(handle.Filename) != ".srt" {
		http.Error(w, "Not a subtitle", 400)
		return
	}

	err = SaveSubtitle(path, file)

	if err != nil {
		http.Error(w, fmt.Sprintf("Error: %s", err), 500)
		return
	}

	json.NewEncoder(w).Encode("OK")
	// mimeType := handle.Header.Get("Content-Type")

}

func watch(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	path := r.URL.Query().Get("path")
	responseEnconder := json.NewEncoder(w)

	if url != "" {
		torrent, err := torrentManager.Download(url)

		if err == nil {
			html := GenerateTorrentVideoPlayer(torrent)
			responseEnconder.Encode(WatchResponse{
				HTML: html,
				Path: torrent.VideoFile.Path(),
			})
		} else {
			responseEnconder.Encode(err)
		}

		return
	}

	if path != "" {
		html := GenerateFileVideoPlayer(path)
		responseEnconder.Encode(WatchResponse{
			HTML: html,
			Path: path,
		})

		return
	}

	http.Error(w, "Not found", 404)
}

func stream(w http.ResponseWriter, r *http.Request) {
	infoHash := r.URL.Query().Get("infoHash")
	path := r.URL.Query().Get("path")

	if infoHash != "" {
		torrent := torrentManager.GetTorrentFromHash(infoHash)

		if torrent != nil {
			StreamTorrent(torrent, w, r)
			return
		}

		http.Error(w, "Not found", 404)
		return
	} else if path != "" {
		file, err := os.Open(path)

		if err == nil {
			StreamFile(file, w, r)
			return
		}

		http.Error(w, "Not found", 404)
		return
	}

	http.Error(w, "Not found", 404)
}

func download(w http.ResponseWriter, r *http.Request) {
	body := GetBody(w, r)

	msg := struct {
		URL string `json:"url"`
	}{}

	err := json.Unmarshal(body, &msg)

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	torrent, err := torrentManager.Download(msg.URL)

	if err == nil {
		go GetSubtitles(torrent.VideoFile.Path())
		json.NewEncoder(w).Encode(torrent)
	} else {
		http.Error(w, err.Error(), 500)
	}
}

// StartWebserver starts the webserver to start listening to HTTP requests
func StartWebServer(mainChannel chan string) {
	log.Print("Starting web server...")
	corsObj := handlers.AllowedOrigins([]string{"*"})
	router := mux.NewRouter()

	router.HandleFunc("/torrents", torrents).Methods("GET")
	router.HandleFunc("/torrents", download).Methods("POST")
	router.HandleFunc("/watch", watch).Methods("GET")
	router.HandleFunc("/stream", stream).Methods("GET")
	router.HandleFunc("/subtitles", subtitles).Methods("GET")
	router.HandleFunc("/subtitles", uploadSubtitles).Methods("POST")

	err := http.ListenAndServe(
		fmt.Sprintf(":%s", viper.GetString("port")),
		handlers.CORS(corsObj)(router),
	)

	if err != nil {
		mainChannel <- err.Error()
	}
}
