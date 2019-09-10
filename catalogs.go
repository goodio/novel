// 目录
package main

import (
	"encoding/json"
	"github.com/PuerkitoBio/goquery"
	"github.com/ghaoo/rbootx/tools"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/sirupsen/logrus"
	"net"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"time"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"fmt"
)

type Catalog struct {
	ID          int       // ID
	SubID       string    // SUB ID
	Name        string    // 名称
	Author      string    // 作者
	Url         string    // 链接
	Chapters    []Chapter // 章节ID列表
	Category    string    // 类别
	LastChapter string    // 最新章节
	LastUpdate  string    // 最后更新
}

type Chapter struct {
	ID   int
	Url  string
	Name string
}

func GetCatalog(url string) Catalog {
	cl := Catalog{}

	c := colly.NewCollector(
		colly.AllowedDomains("www.bqg5200.com"),
	)

	c.Limit(&colly.LimitRule{
		Parallelism: 1,
		RandomDelay: 5 * time.Second,
	})

	c.WithTransport(&http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		//DisableKeepAlives: true,
	})

	extensions.RandomUserAgent(c)

	var reg = regexp.MustCompile(`https:\/\/www.bqg5200.com\/xiaoshuo\/(\d+)\/(\d+)[\/]?$`)
	var reg2 = regexp.MustCompile(`https:\/\/www.bqg5200.com\/xiaoshuo\/\d+\/\d+\/(\d+).html`)
	c.OnHTML("div#maininfo", func(e *colly.HTMLElement) {
		url := e.Request.URL.String()

		idr := reg.FindStringSubmatch(url)

		subid := idr[1]

		idstr := idr[2]

		id, _ := strconv.Atoi(idstr)

		h, _ := e.DOM.Html()

		html, _ := tools.DecodeGBK([]byte(h))

		dom := e.DOM.SetHtml(string(html))

		title := dom.Find("div.coverecom div:nth-of-type(2)")

		name := title.Find("h1").Text()

		author := title.Find("span:first-of-type").Text()

		category := title.Find("span:nth-of-type(2) a").Text()

		last_update := title.Find("span:nth-of-type(3)").Text()

		last_chapter := dom.Find("#readerlist ul li:last-of-type a").Text()

		cpts := []Chapter{}
		dom.Find("#readerlist ul li").Each(func(i int, s *goquery.Selection) {

			cname := s.Find("a").Text()
			curl, _ := s.Find("a").Attr("href")
			curl = e.Request.AbsoluteURL(curl)

			if reg2.MatchString(curl) {
				cid, err := strconv.Atoi(reg2.FindStringSubmatch(curl)[1])

				if err != nil {
					cid = 0
				}

				cpt := Chapter{
					ID:   cid,
					Name: cname,
					Url:  curl,
				}

				cpts = append(cpts, cpt)
			}

		})

		cl.ID = id
		cl.SubID = subid
		cl.Name = name
		cl.Author = author
		cl.Url = url
		cl.Category = category
		cl.Chapters = cpts
		cl.LastChapter = last_chapter
		cl.LastUpdate = last_update

		fname := path.Join(BOOK_PATH, name, "data.json")

		data, err := json.Marshal(&cl)
		if err != nil {
			logrus.Error(err)
		} else {
			tools.FileWrite(fname, data)
		}

	})

	c.Visit(url)

	c.Wait()

	return cl
}

func FetchCatalog() {

	cl := Catalog{}

	c := colly.NewCollector(
		colly.AllowedDomains("www.bqg5200.com"),
	)

	c.Limit(&colly.LimitRule{
		Parallelism: 1,
		RandomDelay: 5 * time.Second,
	})

	c.WithTransport(&http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
			DualStack: true,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		//DisableKeepAlives: true,
	})

	extensions.RandomUserAgent(c)

	var reg = regexp.MustCompile(`https:\/\/www.bqg5200.com\/xiaoshuo\/(\d+)\/(\d+)[\/]?$`)
	var reg2 = regexp.MustCompile(`https:\/\/www.bqg5200.com\/xiaoshuo\/\d+\/\d+\/(\d+).html`)
	c.OnHTML("div#maininfo", func(e *colly.HTMLElement) {
		url := e.Request.URL.String()

		idr := reg.FindStringSubmatch(url)

		subid := idr[1]

		idstr := idr[2]

		id, _ := strconv.Atoi(idstr)

		h, _ := e.DOM.Html()

		html, _ := tools.DecodeGBK([]byte(h))

		dom := e.DOM.SetHtml(string(html))

		title := dom.Find("div.coverecom div:nth-of-type(2)")

		name := strings.Replace(title.Find("h1").Text(), `\`, "", -1)

		author := title.Find("span:first-of-type").Text()

		category := title.Find("span:nth-of-type(2) a").Text()

		last_update := title.Find("span:nth-of-type(3)").Text()

		last_chapter := dom.Find("#readerlist ul li:last-of-type a").Text()

		cpts := []Chapter{}
		dom.Find("#readerlist ul li").Each(func(i int, s *goquery.Selection) {

			cname := strings.Replace(s.Find("a").Text(), `\`, "", -1)
			curl, _ := s.Find("a").Attr("href")
			curl = e.Request.AbsoluteURL(curl)

			if reg2.MatchString(curl) {
				cid, err := strconv.Atoi(reg2.FindStringSubmatch(curl)[1])

				if err != nil {
					cid = 0
				}

				cpt := Chapter{
					ID:   cid,
					Name: cname,
					Url:  curl,
				}

				cpts = append(cpts, cpt)
			}

		})

		cl.ID = id
		cl.SubID = subid
		cl.Name = name
		cl.Author = author
		cl.Url = url
		cl.Category = category
		cl.Chapters = cpts
		cl.LastChapter = last_chapter
		cl.LastUpdate = last_update

		fname := path.Join(BOOK_PATH, name, "data.json")

		data, err := json.Marshal(&cl)
		if err != nil {
			logrus.Error(err)
		} else {
			tools.FileWrite(fname, data)
		}

	})

	c.OnRequest(func(r *colly.Request) {
		logrus.Info("访问地址：", r.AbsoluteURL(r.URL.String()))
	})

	files, _ := filepath.Glob(BOOK_PATH + "\\*")

	exist := make(map[int]int, 0)
	for _, v := range files {

		datafile := filepath.Join(v, "data.json")

		cl = read(datafile)

		for i := 57260; i >= 1; i-- {

			if cl.ID == i {

				exist[i] = i
			}

		}
	}
	//57260
	for i := 39293; i >= 1; i-- {
		if _, ok := exist[i]; !ok {

			id := strconv.Itoa(i)
			subid := "0"

			if len(id) > 3 {
				subid = strings.TrimSuffix(id, id[len(id)-3:])
			}

			href := fmt.Sprintf("https://www.bqg5200.com/xiaoshuo/%s/%s/", subid, id)
			c.Visit(href)
		}
	}

	c.Wait()

}

func read(datafile string) (cl Catalog) {

	data, err := ioutil.ReadFile(datafile)
	if err != nil {
		os.Remove(datafile)
		//logrus.Errorf("读取文件【%s】失败: %v", datafile, err)
		return
	}

	err = json.Unmarshal(data, &cl)
	if err != nil {
		os.Remove(datafile)
		//logrus.Errorf("解析文件【%s】失败: %v", datafile, err)
		return
	}

	return
}

