package tumblr

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	host  = "https://api.tumblr.com"
	limit = 20
)

type TumblrClient struct {
	apikey     string
	blogname   string
	imageUrls  []string
	idCapacity int
	rand       *rand.Rand
}

type ResponsePhotoPosts struct {
	Meta struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	} `json:"meta"`
	Response struct {
		Blog struct {
			Title       string `json:"title"`
			Name        string `json:"name"`
			Posts       int    `json:"posts"`
			Url         string `json:"url"`
			Updated     int    `json:"updated"`
			Description string `json:"description"`
			Ask         bool   `json:"ask"`
			AskAnon     bool   `json:"ask_anon"`
			IsNsfw      bool   `json:"is_nsfw"`
			ShareLikes  bool   `json:"share_likes"`
		} `json:"blog"`
		Posts []struct {
			Blogname       string   `json:"blog_name"`
			Id             int      `json:"id"`
			PostUrl        string   `json:"post_url"`
			Slug           string   `json:"slug"`
			Type           string   `json:"type"`
			Date           string   `json:"date"`
			Timestamp      int      `json:"timestamp"`
			State          string   `json:"state"`
			Format         string   `json:"format"`
			ReblogKey      string   `json:"reblog_key"`
			Tags           []string `json:"tags"`
			ShortUrl       string   `json:"short_url"`
			NoteCount      int      `json:"note_count"`
			SourceUrl      string   `json:"source_url"`
			SourceTitle    string   `json:"source_title"`
			Caption        string   `json:"caption"`
			ImagePermalink string   `json:"image_permalink"`
			Photos         []struct {
				Caption  string `json:"caption"`
				AltSizes []struct {
					Width  int    `json:"width"`
					Height int    `json:"height"`
					Url    string `json:"url"`
				} `json:"alt_sizes"`
				OriginalSize struct {
					Width  int    `json:"width"`
					Height int    `json:"height"`
					Url    string `json:"url"`
				} `json:"original_size"`
			} `json:"photos"`
		} `json:"posts"`
		TotalPosts int `json:"total_posts"`
	} `json:"response"`
}

func NewTumblrClient(blogname string, apikey string, capacity int) *TumblrClient {
	client := &TumblrClient{}
	client.apikey = apikey
	client.imageUrls = make([]string, 0)
	client.idCapacity = capacity
	client.blogname = blogname
	client.rand = rand.New(rand.NewSource(time.Now().Unix()))

	return client
}

func (client *TumblrClient) StartCaching() {

	c := make(chan string)
	go func() {
		offset := 0
		for !client.IsStoped() {
			res, err := client.Search(strconv.Itoa(offset), strconv.Itoa(limit))

			if err != nil {
				fmt.Println(err)
				close(c)
				break
			}
			if len(res.Response.Posts) < 1 {
				fmt.Println("posts count is 0\n")
				close(c)
				break
			}

			for _, post := range res.Response.Posts {
				for _, photo := range post.Photos {
					c <- photo.OriginalSize.Url
				}
			}
			offset += limit
			time.Sleep(time.Second)
		}
	}()

	for {
		url, ok := <-c
		if !ok {
			break
		}
		fmt.Println(url, "\n")
		client.imageUrls = append(client.imageUrls, url)
	}
}

func (client *TumblrClient) Search(limit string, offset string) (*ResponsePhotoPosts, error) {
	v := url.Values{}
	v.Set("api_key", client.apikey)
	v.Set("limit", limit)
	v.Set("offset", offset)

	resp, err := http.Get(fmt.Sprintf("%s/v2/blog/%s/posts/photo?%s", host, client.blogname, v.Encode()))

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var ret ResponsePhotoPosts
	json.Unmarshal(body, &ret)

	return &ret, nil
}

func (client *TumblrClient) IsStoped() bool {
	return len(client.imageUrls) >= client.idCapacity
}

func (client *TumblrClient) Run() {
	client.StartCaching()
}

func (client *TumblrClient) GetImage() ([]byte, error) {

	if len(client.imageUrls) < 0 {
		time.Sleep(time.Second * 3)
	}

	if len(client.imageUrls) < 1 {
		return nil, errors.New("empty image urls")
	}

	index := client.rand.Int31n((int32)(len(client.imageUrls)))
	url := client.imageUrls[index]

	resp, err := http.Get(url)

	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	return body, err
}
