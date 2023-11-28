package main

import (
	"context"
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
	"time"
	"vrc-daily-uploader/flickrapi"
)

const TimeoutSeconds = 3

func config() (flickrapi.Config, error) {
	var conf flickrapi.Config
	conf.ApiKey = os.Getenv("API_KEY")
	conf.UserId = os.Getenv("USER_ID")
	conf.SearchEndPoint = "https://www.flickr.com/services/rest/"
	conf.SearchMethod = "flickr.photos.search"
	conf.ImageEndPoint = "https://live.staticflickr.com"
	return conf, nil
}

func getPhotosSearchWithContext(ctx context.Context, conf flickrapi.Config, pageNum string, ch chan<- flickrapi.PhotosSearchJson, wg *sync.WaitGroup) {
	defer wg.Done()

	select {
	case <-ctx.Done():
		log.Fatal("error: Connection timeout")
	default:
		searchJson, err := getPhotosSearch(conf, pageNum)
		if err != nil {
			log.Fatal(err)
		}
		ch <- searchJson
	}
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

	//fmt.Println(u.String())

	resp, err := http.Get(u.String())
	if err != nil {
		return flickrapi.PhotosSearchJson{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return flickrapi.PhotosSearchJson{}, fmt.Errorf("error: Response status is %d", resp.StatusCode)
	}

	var photosSearchJson flickrapi.PhotosSearchJson
	err = json.NewDecoder(resp.Body).Decode(&photosSearchJson)
	if err != nil {
		return flickrapi.PhotosSearchJson{}, err
	}

	return photosSearchJson, nil
}

func SavePhotoWithContext(ctx context.Context, conf flickrapi.Config, photoList flickrapi.Photo, num int) error {
	select {
	case <-ctx.Done():
		log.Fatal("error: Connection timeout")
	default:
		err := SavePhoto(conf, photoList, num)
		if err != nil {
			return err
		}
	}
	return nil
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
	file, err := os.Create("./image" + strconv.Itoa(num) + ".jpg")
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
	ctx, cancel := context.WithTimeout(context.Background(), TimeoutSeconds*time.Second)
	//defer cancel()

	for i := 1; i < 3; i++ {
		wg.Add(1)
		go getPhotosSearchWithContext(ctx, conf, strconv.Itoa(i), ch, &wg)
	}

	go func() {
		wg.Wait()
		close(ch)
		cancel() //再利用するのでここでcancel()
	}()

	for json := range ch {
		photoList = append(photoList, json.Photos.PhotoList...)
	}
	//fmt.Println(photoList)

	var mu sync.Mutex
	ctx, cancel = context.WithTimeout(context.Background(), TimeoutSeconds*time.Second)
	defer cancel()

	for i := 1; i < 4; i++ {
		wg.Add(1)
		tmp := rand.Intn(len(photoList))
		go func(photo []flickrapi.Photo, tmp int, idx int) {
			defer wg.Done()
			errCnt := 0

			for {
				err := SavePhotoWithContext(ctx, conf, photoList[tmp], idx)
				if err != nil && errCnt < 3 {
					errCnt++
					if 3 <= errCnt {
						fmt.Println(err)
						break
					}
				} else {
					mu.Lock()
					photoList = removeEl(photoList, tmp)
					mu.Unlock()
					break
				}
			}
		}(photoList, tmp, i)
	}
	wg.Wait()
}
