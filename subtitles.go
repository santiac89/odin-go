package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/oz/osdb"
	"github.com/spf13/viper"
	"github.com/umahmood/subdb"
)

const MAX_SUBS_PER_LANG = 5

func GetSubtitles(path string) (s []string) {
	dir := filepath.Dir(path)

	log.Print("Looking for subs in ", dir)

	files, err := ioutil.ReadDir(dir)

	if err != nil {
		log.Print("Failed to read dir ", err)
	} else {
		currentSubs := make([]string, 0)

		for _, file := range files {
			if filepath.Ext(file.Name()) == ".srt" {
				subtitlePath, err := filepath.Abs(filepath.Join(dir, file.Name()))

				if err == nil {
					currentSubs = append(currentSubs, subtitlePath)
				}
			}
		}

		if len(currentSubs) > 0 {
			log.Print("Subs found! 🤘")
			return currentSubs
		}
	}

	log.Print("Subs not found, looking on THE INTERNET ...")

	return DownloadSubtitlesOSDB(path)
}

func DownloadSubtitlesSubdb(path string) (f []string) {
	finalSubs := make([]string, 0)

	subdb := &subdb.API{}
	subdb.SetUserAgent("Acme", "1.0", "https://acme.org")

	languages := viper.GetStringSlice("subdb_langs")

	log.Print(fmt.Sprintf("Searching %s subtitles in SubDB for %s", languages, path))

	absolutePath, err := filepath.Abs(path)

	if err != nil {
		log.Panic(err)
		return finalSubs
	}

	extension := filepath.Ext(path)
	subtitleFilenameBase := strings.Replace(absolutePath, extension, "", 0)

	for _, lang := range languages {
		subtitleContent, err := subdb.Download(absolutePath, lang)

		if err != nil {
			fmt.Println(err)
			continue
		}

		subtitleFilename := subtitleFilenameBase + ".subdb." + lang + ".srt"
		subtitleFile, err := os.Create(subtitleFilename)

		if err != nil {
			log.Panic(err)
			continue
		}

		subtitleFile.WriteString(subtitleContent)
		subtitleFile.Close()

		finalSubs = append(finalSubs, subtitleFilename)
	}

	return finalSubs
}

func DownloadSubtitlesOSDB(path string) (f []string) {
	finalSubs := make([]string, 0)
	client, err := osdb.NewClient()

	if err != nil {
		log.Print("Couldn't initialize OSDB client", err)
		return finalSubs
	}

	err = client.LogIn(viper.GetString("os_username"), viper.GetString("os_password"), "es,en")

	if err != nil {
		log.Print("Couldn't login to OSDB", err)
		return finalSubs
	}

	languages := viper.GetStringSlice("os_langs")

	absolutePath, err := filepath.Abs(path)

	if err != nil {
		log.Panic(err)
		return finalSubs
	}

	subtitleFilenameBase := strings.Replace(absolutePath, filepath.Ext(path), "", 0)

	log.Print(fmt.Sprintf("Searching %s subtitles in OSDB for %s", languages, absolutePath))

	params := []interface{}{
		client.Token,
		[]struct {
			Query string `xmlrpc:"query"`
			Langs string `xmlrpc:"sublanguageid"`
		}{{
			filepath.Base(path),
			strings.Join(languages, ","),
		}},
	}

	subtitles, err := client.SearchSubtitles(&params)

	if err != nil {
		log.Print("Couldn't search for subs in OSDB", err)
		return finalSubs
	}

	if len(subtitles) == 0 {
		log.Print("No subs found in OSDB")
		return finalSubs
	}

	subsByLang := partitionByLang(subtitles)

	for lang := range subsByLang {
		for i, subtitle := range subsByLang[lang] {
			subtitleFilename := fmt.Sprintf("%s.osdb.%s.%d.srt", subtitleFilenameBase, subtitle.LanguageName, i)
			log.Print("Downloading ", subtitleFilename)
			err := client.DownloadTo(&subtitle, subtitleFilename)

			if err != nil {
				log.Print(err)
				continue
			}

			finalSubs = append(finalSubs, subtitleFilename)

			if i == MAX_SUBS_PER_LANG {
				break
			}
		}
	}

	return finalSubs
}

func partitionByLang(subtitles osdb.Subtitles) (res map[string]osdb.Subtitles) {
	m := make(map[string]osdb.Subtitles)

	for _, item := range subtitles {
		_, ok := m[item.LanguageName]

		if !ok {
			m[item.LanguageName] = osdb.Subtitles{}
		}

		m[item.LanguageName] = append(m[item.LanguageName], item)
	}

	return m
}

func SaveSubtitle(path string, file multipart.File) error {
	data, err := ioutil.ReadAll(file)

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(viper.GetString("download_dir")+"/"+path+".user.User.0.srt", data, 0666)

	if err != nil {
		return err
	}

	return nil
}
