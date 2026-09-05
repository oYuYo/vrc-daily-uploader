package main

import (
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
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
		log.Fatalf("error: photo search timed out: %v", ctx.Err())
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

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return flickrapi.PhotosSearchJson{}, err
	}
	req.Header.Set("User-Agent", "vrc-daily-uploader/1.0 (+https://github.com/oYuYo/vrc-daily-uploader)")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return flickrapi.PhotosSearchJson{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body)
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

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "vrc-daily-uploader/1.0 (+https://github.com/oYuYo/vrc-daily-uploader)")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		_, _ = io.Copy(io.Discard, resp.Body) //bodyを読み捨てると, connectionが閉じずに再利用される
		return fmt.Errorf("error: image request to %s returned status %d", u, resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return fmt.Errorf("error: failed to decode image from %s: %w", u, err)
	}

	var rotatedImg image.Image
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	if isHorizontal(width, height) {
		rotatedImg = img
	} else {
		rotatedImg = rotateToHorizonal(img)
	}

	file, err := os.Create("./image" + strconv.Itoa(num) + ".jpg")
	if err != nil {
		return err
	}
	defer file.Close()

	if err := saveImageToFile(rotatedImg, file); err != nil {
		return err
	}

	return nil
}

func isHorizontal(width int, height int) bool {
	flg := false
	if width > height {
		flg = true
	}
	return flg
}

// 縦向きの写真を横向きへ, 背景は白で塗りつぶす(w:h = 10:7 で表示しているため)
func rotateToHorizonal(img image.Image) image.Image {
	newWidth := int(float64(img.Bounds().Dy()) / 0.7)
	newHeight := img.Bounds().Dy()
	resizedImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	draw.Draw(resizedImg, resizedImg.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Over)

	centerX := (newWidth - img.Bounds().Dx()) / 2
	centerY := (newHeight - img.Bounds().Dy()) / 2

	draw.Draw(resizedImg, img.Bounds().Add(image.Pt(centerX, centerY)), img, image.Point{}, draw.Over)

	return resizedImg
}

func saveImageToFile(img image.Image, file io.Writer) error {
	return jpeg.Encode(file, img, nil)
}

// 指定したindexを削除したスライスを返す
func removeEl(photoList []flickrapi.Photo, num int) []flickrapi.Photo {
	return append(photoList[:num], photoList[num+1:]...)
}

func sleepOrCancel(ctx context.Context, duration time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(duration):
		return nil
	}
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
	if len(photoList) == 0 {
		log.Fatal("photo list is empty")
	}
	//fmt.Println(photoList)

	ctx, cancel = context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for i := 1; i < 4; i++ {
		wg.Add(1)
		tmp := rand.Intn(len(photoList))

		go func(photo flickrapi.Photo, idx int) {
			defer wg.Done()
			var err error
			const maxAttempts = 3
			for attempt := 0; attempt < maxAttempts; attempt++ {
				err = SavePhotoWithContext(ctx, conf, photo, idx)
				if err == nil {
					return
				}
				if attempt < maxAttempts-1 {
					if waitErr := sleepOrCancel(ctx, time.Duration(attempt+1)*time.Second); waitErr != nil {
						err = fmt.Errorf("%w (after attempt %d)", waitErr, attempt+1)
					}
				}
			}
			fmt.Println(err)
		}(photoList[tmp], i)
		photoList = removeEl(photoList, tmp)
	}
	wg.Wait()
}
