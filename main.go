package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Referer

const articlesPage = "http://apipc.app.acfun.cn/articles/"

var (
	itemExists = struct{}{}
	regImage   = regexp.MustCompile("http://(.*?)jpg|http://(.*?)png|http://(.*?)jpeg")
	header     = map[string][]string{
		"User-Agent": []string{"Mozilla/5.0 (Windows NT 10.0; rv:56.0) Gecko/20100101 Firefox/56.0"},
		"deviceType": []string{"1"},
	}
	articleID string
	referer   = "http://www.acfun.cn/a/"
)

func init() {
	flag.StringVar(&articleID, "id", "", "文章id")
	flag.Parse()

	if articleID == "" {
		flag.Usage()
		os.Exit(-1)
	}

	// 组成 referer
	referer += articleID
}

// Article json文章结构
type Article struct {
	Data struct {
		Article struct {
			Content string `json:"content,omitempty"`
		} `json:"article,omitempty"`
		Title string `json:"title,omitempty"`
	} `json:"data,omitempty"`
}

// getImageURL 获取正则匹配出的图片url
func (a *Article) getImageURL() []string {
	uniq := make(map[string]struct{})
	matchs := regImage.FindAllString(a.Data.Article.Content, -1)
	result := make([]string, 0, len(matchs))

	// 用 map 去重
	for _, v := range matchs {
		uniq[v] = itemExists
	}
	for v := range uniq {
		result = append(result, v)
	}
	return result
}

// getArticle
func getArticle(contentID string) *Article {
	var article Article
	contentID = strings.TrimPrefix(contentID, "ac")

	req, err := http.NewRequest(http.MethodGet, articlesPage+contentID, nil)
	if err != nil {
		log.Fatalln(err)
	}
	req.Header = header

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()

	doc, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}

	if err := json.Unmarshal(doc, &article); err != nil {
		log.Fatalln(err)
	}

	return &article
}

func download(c chan int, target, root string) {
	_, fileName := filepath.Split(target)
	fileName = filepath.Join(root, fileName)
	if existFile(fileName) {
		return
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		return
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; rv:56.0) Gecko/20100101 Firefox/56.0")
	req.Header.Set("Referer", referer)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatalln(err)
	}
	defer resp.Body.Close()

	file, err := os.Create(fileName)
	defer file.Close()

	if err != nil {
		log.Fatalln(err)
	}

	io.Copy(file, resp.Body)
	println(fileName, "Done!")
	<-c
}

func existFile(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}

func distributed(urls []string, root string) {
	c := make(chan int, 2)

	for _, v := range urls {
		c <- 1
		download(c, v, root)
		time.Sleep(1 * time.Second)
	}
}

func main() {
	root := filepath.Join(".", articleID)
	os.Mkdir(root, os.ModePerm)

	artilce := getArticle(articleID)
	distributed(artilce.getImageURL(), root)
}
