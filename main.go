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

func getPhotosSearch(conf flickrapi.Config, pageNum string) (flickrapi.PhotosSearchJson, error) {
	u, err := url.Parse(conf.SearchEndPoint)
	if err != nil {
		return flickrapi.PhotosSearchJson{}, err
	}

	q := u.Query()
	q.Set("method", conf.SearchMethod)
	q.Set("api_key", conf.ApiKey)
	q.Set("user_id", conf.UserId)
	q.Set("page", pageNum)
	q.Set("format", "json")
	q.Set("nojsoncallback", "1")
	u.RawQuery = q.Encode()

	fmt.Println(u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		return flickrapi.PhotosSearchJson{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return flickrapi.PhotosSearchJson{}, err
	}
	var photosSearchJson flickrapi.PhotosSearchJson
	err = json.NewDecoder(resp.Body).Decode(&photosSearchJson)
	if err != nil {
		return flickrapi.PhotosSearchJson{}, err
	}

	return photosSearchJson, nil
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
	fmt.Println(u)

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

func main() {
	conf, err := config()
	if err != nil {
		log.Fatal(err)
	}

	var photosSearchJson flickrapi.PhotosSearchJson
	var photoList []flickrapi.Photo

	for i := 1; i < 3; i++ {
		photosSearchJson, err = getPhotosSearch(conf, strconv.Itoa(i))
		if err != nil {
			log.Fatal(err)
		}
		photoList = append(photoList, photosSearchJson.Photos.PhotoList...)
	}

	var t int
	i := 0
	for i < 3 {
		tmp := rand.Intn(len(photoList) + 1)
		if t != tmp {
			t = tmp
			fmt.Println(t)
			SavePhoto(conf, photoList[t-1], i+1)
			i++
		}
	}
}
