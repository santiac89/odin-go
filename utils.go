package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/anacrolix/torrent"
	"github.com/spf13/viper"
)

// DownloadFile it downloads a file to a temporary directory, it receives the URL for the file as parameter.
// Parameters:
// URL string -> The URL of the file to download
func DownloadFile(URL string) (fileName string, err error) {
	var file *os.File
	if file, err = ioutil.TempFile(os.TempDir(), "odin"); err != nil {
		return
	}

	defer func() {
		if ferr := file.Close(); ferr != nil {
			log.Printf("Error closing torrent file: %s", ferr)
		}
	}()

	response, err := http.Get(URL)
	if err != nil {
		return
	}

	defer func() {
		if ferr := response.Body.Close(); ferr != nil {
			log.Printf("Error closing torrent file: %s", ferr)
		}
	}()

	_, err = io.Copy(file, response.Body)

	return file.Name(), err
}

// StreamTorrent starts streaming the video being downloaded by a torrent.
// Parameters:
// torr *torrent.Torrent -> The torrent to stream
// response http.ResponseWriter -> The response where to stream
// request *http.Request -> The request that requested the stream
func StreamTorrent(torr *torrent.Torrent, response http.ResponseWriter, request *http.Request) {
	var videoFile *torrent.File

	for _, file := range torr.Files() {
		if strings.HasSuffix(file.Path(), "mp4") {
			videoFile = file
		}
	}

	reader := videoFile.NewReader()

	reader.SetReadahead(videoFile.Length() / 100)
	reader.SetResponsive()
	reader.Seek(videoFile.Offset(), os.SEEK_SET)

	http.ServeContent(response, request, videoFile.DisplayPath(), time.Now(), reader)
}

// StreamFile starts streaming a video file located in disk
// Parameters:
// file *os.File -> The file to stream
// response http.ResponseWriter -> The response where to stream
// request *http.Request -> The request that requested the stream
func StreamFile(file *os.File, response http.ResponseWriter, request *http.Request) {
	http.ServeContent(response, request, file.Name(), time.Now(), file)
}

// GenerateFileVideoPlayer responds to the HTTP request with the HTML videoplayer tag using a torrent being downloaded as a source
// Parameters:
// torr *TorrentInfo -> The struct containing the downloading torrent information
// response http.ResponseWriter -> The response where to send the HTML
// request *http.Request -> The request
func GenerateTorrentVideoPlayer(torrentInfo *TorrentInfo) string {
	port := viper.GetString("port")
	infoHash := torrentInfo.InfoHash
	subtitlesTag := GenerateSubtitlesTags(torrentInfo.VideoFile.Path())

	// response.Header().Add("Content-Type", "text/html")

	html := `<video class="player" crossorigin="anonymous" controls>
						<source src="http://localhost:%s/stream?infoHash=%s" type="video/webm">
						%s
					</video>`

	return fmt.Sprintf(html, port, infoHash, subtitlesTag)
}

// GenerateFileVideoPlayer responds to the HTTP request with the HTML videoplayer tag using a file as source of the video.
// It receives the path of the file, the response and the request as parameters.
func GenerateFileVideoPlayer(path string) string {
	port := viper.GetString("port")
	subtitlesTag := GenerateSubtitlesTags(path)

	// response.Header().Add("Content-Type", "text/html")

	html := `<video class="player" crossorigin="anonymous" controls>
						<source src="http://localhost:%s/stream?path=%s" type="video/webm">
						%s
					</video>`

	return fmt.Sprintf(html, port, path, subtitlesTag)
}

// GenerateSubtitlesTags receives a path as argument and in case of success returns the <track> tags for the HTML videoplayer with the subtitles found
// Parameters:
// videoFilePath string -> The video file path where to search for subtitles
func GenerateSubtitlesTags(videoFilePath string) (r string) {
	port := viper.GetString("port")
	subtitles := GetSubtitles(videoFilePath)
	regexp := regexp.MustCompile("(subdb|osdb|user)\\.([a-zA-z]+)\\.([0-9]+)\\.srt")
	subtitlesTag := ""

	for _, subtitle := range subtitles {
		matches := regexp.FindStringSubmatch(subtitle)
		if len(matches) == 0 {
			continue
		}

		subtitlesTag += fmt.Sprintf(`<track src="http://localhost:%s/subtitles?path=%s" kind="subtitles" srclang="%s" />`, port, subtitle, matches[2]+"-"+matches[1]+"-"+matches[3])
	}

	return subtitlesTag
}

// GetBody reads the body of a POST request and responds with an error in case of failure
func GetBody(w http.ResponseWriter, r *http.Request) (res []byte) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()

	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	return b
}
