package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"sync"
	"vrc-daily-uploader/flickrapi"
)

func config() (flickrapi.Config, error) {
	var conf flickrapi.Config
	conf.ApiKey = os.Getenv("API_KEY")
	conf.UserId = os.Getenv("USER_ID")
	conf.SearchEndPoint = "https://www.flickr.com/services/rest/"
	conf.SearchMethod = "flickr.photos.search"
	conf.ImageEndPoint = "https://live.staticflickr.com"
	return conf, nil
}

func getPhotosSearch(conf flickrapi.Config, pageNum string, ch chan<- flickrapi.PhotosSearchJson, wg *sync.WaitGroup) {
	defer wg.Done()

	u, err := url.Parse(conf.SearchEndPoint)
	if err != nil {
		ch <- flickrapi.PhotosSearchJson{}
		return
	}

	q := u.Query()
	q.Set("method", conf.SearchMethod)
	q.Set("api_key", conf.ApiKey)
	q.Set("user_id", conf.UserId)
	q.Set("page", pageNum)
	q.Set("format", "json")
	q.Set("nojsoncallback", "1")
	u.RawQuery = q.Encode()

	//fmt.Println(u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		fmt.Println(err)
		ch <- flickrapi.PhotosSearchJson{}
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.StatusCode)
		ch <- flickrapi.PhotosSearchJson{}
		return
	}

	var photosSearchJson flickrapi.PhotosSearchJson
	err = json.NewDecoder(resp.Body).Decode(&photosSearchJson)
	if err != nil {
		fmt.Println(err)
		ch <- flickrapi.PhotosSearchJson{}
		return
	}

	ch <- photosSearchJson
}

func SavePhoto(conf flickrapi.Config, photoList flickrapi.Photo, num int) error {
	serverId := photoList.Server
	id := photoList.Id
	secret := photoList.Secret
	sizeSuffix := "b"

	u := conf.ImageEndPoint
	u, err := url.JoinPath(u, serverId, id+"_"+secret+"_"+sizeSuffix+".jpg")
	if err != nil {
		return err
	}
	//fmt.Println(u)

	resp, err := http.Get(u)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	file, err := os.Create("./data/image" + strconv.Itoa(num) + ".jpg")
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return err
	}
	return nil
}

// 指定したindexを削除したスライスを返す
func removeEl(photoList []flickrapi.Photo, num int) []flickrapi.Photo {
	return append(photoList[:num], photoList[num+1:]...)
}

func main() {
	conf, err := config()
	if err != nil {
		log.Fatal(err)
	}

	var photoList []flickrapi.Photo
	var wg sync.WaitGroup
	ch := make(chan flickrapi.PhotosSearchJson)
	for i := 1; i < 3; i++ {
		wg.Add(1)
		go getPhotosSearch(conf, strconv.Itoa(i), ch, &wg)

	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	for json := range ch {
		photoList = append(photoList, json.Photos.PhotoList...)
	}
	//fmt.Println(photoList)

	i := 0
	for i < 3 {
		tmp := rand.Intn(len(photoList) + 1)
		SavePhoto(conf, photoList[tmp-1], i+1)
		photoList = removeEl(photoList, tmp)
		i++
	}
}
