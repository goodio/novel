package main

import (
	"github.com/ghaoo/rbootx"
	"github.com/ghaoo/rbootx/tools"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/sirupsen/logrus"

	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"io/ioutil"
	"encoding/json"
)

var getreg = regexp.MustCompile(`#小说 (.+)`)

func GetBook(bot *rbootx.Robot) []rbootx.Message {

	var msg rbootx.Message

	book := ""
	in := bot.Incoming()
	if getreg.MatchString(in.Content) {
		bs := getreg.FindStringSubmatch(in.Content)
		book = bs[1]
	}

	if book != "" {
		fname := filepath.Join(BOOK_PATH, book)

		if _, err := os.Stat(fname); err != nil {

			if os.IsNotExist(err) {
				// 不存在
				return []rbootx.Message{
					{
						Content: "没有找到 《" + book + "》 这本书",
					},
				}
			}
		} else {
			// 存在
			datafile := filepath.Join(fname, "data.json")

			logrus.Warn(book)

			cl := Catalog{}

			data, err := ioutil.ReadFile(datafile)
			if err != nil {
				logrus.Errorf("读取文件【%s】失败: %v", datafile, err)
			}

			err = json.Unmarshal(data, &cl)
			if err != nil {
				logrus.Errorf("解析文件【%s】失败: %v", datafile, err)
			}

			cl = GetCatalog(fmt.Sprintf("https://www.bqg5200.com/xiaoshuo/%s/%d/", cl.SubID, cl.ID))

			msg.Content = fmt.Sprintf("最新章节：%s", cl.LastChapter)
			bot.Send(msg)

			go fetchContent(&cl)

			if err = fileMerge(fname); err != nil {
				logrus.Error(err)
			}

			bookpath := filepath.Join(fname, book+".txt")
			to := bot.Incoming().To
			logrus.Warn(bookpath,"\n", to)
			if err = bot.SendFile(bookpath, to); err != nil {
				logrus.Error(err)
			}

		}
	} else {
		msg.Content = "别乱来..."
		bot.Send(msg)
	}

	return nil
}

func fetchContent(cl *Catalog) {

	c := colly.NewCollector(
		colly.AllowedDomains("www.bqg5200.com"),
		//colly.DisallowedURLFilters(regexp.MustCompile(`https:\/\/m.bqg5200.com\/wapbook-753-(\d+)*`)),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainRegexp: "www.bqg5200.com/*",
		Parallelism:  30,
		RandomDelay:  5 * time.Second,
	})

	c.WithTransport(&http.Transport{
		DisableKeepAlives: true,
	})

	extensions.RandomUserAgent(c)

	var reg = regexp.MustCompile(`https:\/\/www.bqg5200.com\/xiaoshuo\/\d+\/\d+\/(\d+).html`)

	c.OnHTML("body.clo_bg", func(e *colly.HTMLElement) {

		upath := e.Request.URL.String()

		fname := reg.FindStringSubmatch(upath)

		h, _ := e.DOM.Html()

		html, _ := tools.DecodeGBK([]byte(h))

		dom := e.DOM.SetHtml(string(html))

		class_name := dom.Find("#header .readNav :nth-child(2)").Text()

		book_name := dom.Find("#header .readNav :nth-child(3)").Text()

		title := strings.TrimSpace(dom.Find("div.title h1").Text())

		dom.Find("div#content div").Remove()
		article, _ := dom.Find("div#content").Html()
		article = strings.Replace(article, "聽", " ", -1)
		article = strings.Replace(article, "<br/>", "\n", -1)

		content := "### " + title + "\n" + article + "\n\n"

		fpath := filepath.Join(class_name, book_name, fname[1] + ".rbx")

		err := tools.FileWrite(fpath, []byte(content))

		if err != nil {
			logrus.Errorf("%v\n", err)
		}

	})

	c.OnRequest(func(r *colly.Request) {
		//time.Sleep(getRandomDelay(1000))
		//logrus.Infof("Visiting %s", r.URL.String())
	})

	for _, cpt := range cl.Chapters {

		c.Visit(cpt.Url)
	}

	c.Wait()

}

func fileMerge(root string) error {
	name := filepath.Base(root)

	out_name := filepath.Join(root, name+".txt")

	out_file, err := os.OpenFile(out_name, os.O_CREATE|os.O_WRONLY, 0777)

	if err != nil {
		return fmt.Errorf("Can not open file %s", out_name)
	}

	bWriter := bufio.NewWriter(out_file)

	bWriter.Write([]byte("## " + name + "\n\n\n"))

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

		if !info.IsDir() && strings.HasSuffix(path, ".rbx") {
			//logrus.Printf("读取文件：%s \n", info.Name())

			fp, err := os.Open(path)

			if err != nil {
				fmt.Printf("Can not open file %v", err)
				return err
			}

			defer fp.Close()

			bReader := bufio.NewReader(fp)

			for {

				buffer := make([]byte, 1024)
				readCount, err := bReader.Read(buffer)
				if err == io.EOF {
					break
				} else {
					bWriter.Write(buffer[:readCount])
				}

			}

			bWriter.Write([]byte("\n\n"))
		}

		return err
	})

	bWriter.Flush()

	return nil
}
