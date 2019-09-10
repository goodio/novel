package main

import (
	"github.com/ghaoo/novel/wechat"
	"github.com/go-gomail/gomail"
	"github.com/gocolly/colly"
	"github.com/gocolly/colly/extensions"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/encoding/simplifiedchinese"
	"github.com/patrickmn/go-cache"

	"bufio"
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"sort"
	"strconv"
	"encoding/json"
)

var getreg = regexp.MustCompile(`#(.+)#(\s(?:[a-z0-9_\.-]+)@(?:[\da-z\.-]+)\.(?:[a-z\.]{2,6})$)?`)

var cac = cache.New(10*time.Second, 5*time.Second)

func GetBook(bot *wechat.WeChat, msg wechat.EventMsgData) {

	if msg.AtMe {
		email := ""
		book := ""
		if getreg.MatchString(msg.Content) {
			bs := getreg.FindStringSubmatch(msg.Content)

			book = bs[1]
			email = bs[2]
		}

		if book != "" {

			/*if b, ok := cac.Get(msg.FromUserName); ok {
				bot.SendTextMsg("请检查邮箱是否收到小说" + b.(string), msg.FromUserName)
			}*/

			fname := filepath.Join(BOOK_PATH, book)

			if _, err := os.Stat(fname); err != nil {

				if os.IsNotExist(err) {
					// 不存在
					bot.SendTextMsg("没有找到 《"+book+"》 这本书", msg.FromUserName)
				}
			} else {
				// 存在
				datafile := filepath.Join(fname, "data.json")

				cl := Catalog{}

				data, err := ioutil.ReadFile(datafile)
				if err != nil {
					logrus.Errorf("读取文件【%s】失败: %v", datafile, err)
				}

				err = json.Unmarshal(data, &cl)
				if err != nil {
					logrus.Errorf("解析文件【%s】失败: %v", datafile, err)
				}

				bookpath := filepath.Join(fname, book+".txt")

				if _, err := os.Stat(bookpath); os.IsNotExist(err) {

					bot.SendTextMsg("下载中，请等待几分钟后再来...", msg.FromUserName)

					cl = GetCatalog(fmt.Sprintf("https://www.bqg5200.com/xiaoshuo/%s/%d/", cl.SubID, cl.ID))

					fetchContent(&cl)

					if err = fileMerge(fname); err != nil {
						logrus.Error(err)
					}

					if err = bot.SendFile(bookpath, msg.FromUserName); err != nil {
						if email == "" {
							bot.SendTextMsg("文件较大，需通过邮件发送，请在小说名后面加上邮箱...", msg.FromUserName)
						} else {
							sendmail(email, bookpath, book)
							cac.Set(msg.FromUserName, book, 10*time.Second)
							bot.SendTextMsg("文件较大，已通过邮件发送...", msg.FromUserName)
						}
					}

				} else {

					if err = bot.SendFile(bookpath, msg.FromUserName); err != nil {
						if email == "" {
							bot.SendTextMsg("文件较大，需通过邮件发送，请在小说名后面加上邮箱...", msg.FromUserName)
						} else {
							sendmail(email, bookpath, book)
							cac.Set(msg.FromUserName, book, 10*time.Second)
							bot.SendTextMsg("文件较大，已通过邮件发送...", msg.FromUserName)
						}
					}
				}
			}
		}
	}
}

func fetchContent(cl *Catalog) {

	cac.Set(cl.Name, true, 30*time.Second)

	c := colly.NewCollector(
		colly.AllowedDomains("www.bqg5200.com"),
		//colly.DisallowedURLFilters(regexp.MustCompile(`https:\/\/m.bqg5200.com\/wapbook-753-(\d+)*`)),
		colly.Async(true),
	)

	c.Limit(&colly.LimitRule{
		DomainRegexp: "www.bqg5200.com/*",
		Parallelism:  30,
		RandomDelay:  2 * time.Second,
	})

	/*c.WithTransport(&http.Transport{
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
	})*/

	extensions.RandomUserAgent(c)

	var reg = regexp.MustCompile(`https:\/\/www.bqg5200.com\/xiaoshuo\/\d+\/\d+\/(\d+).html`)

	c.OnHTML("body.clo_bg", func(e *colly.HTMLElement) {

		upath := e.Request.URL.String()

		fname := reg.FindStringSubmatch(upath)

		h, _ := e.DOM.Html()

		html, _ := DecodeGBK([]byte(h))

		dom := e.DOM.SetHtml(string(html))

		//class_name := dom.Find("#header .readNav :nth-child(2)").Text()

		book_name := dom.Find("#header .readNav :nth-child(3)").Text()

		title := strings.TrimSpace(dom.Find("div.title h1").Text())

		dom.Find("div#content div").Remove()
		article, _ := dom.Find("div#content").Html()
		article = strings.Replace(article, "聽", " ", -1)
		article = strings.Replace(article, "<br/>", "\n", -1)

		content := "### " + title + "\n" + article + "\n\n"

		fpath := filepath.Join(BOOK_PATH, book_name, fname[1]+".rbx")

		err := write(fpath, []byte(content))

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

	cac.Delete(cl.Name)

}

func fetchAllContent(root string) {

	files, _ := filepath.Glob(root + "\\*")

	for _, v := range files {

		book := filepath.Join(v, filepath.Base(v)+".txt")

		if _, err := os.Stat(book); os.IsNotExist(err) {
			datafile := filepath.Join(v, "data.json")

			cl := Catalog{}

			data, err := ioutil.ReadFile(datafile)
			if err != nil {
				logrus.Errorf("读取文件【%s】失败: %v", datafile, err)
			}

			err = json.Unmarshal(data, &cl)
			if err != nil {
				logrus.Errorf("解析文件【%s】失败: %v", datafile, err)
			}

			fmt.Printf("开始处理文件夹：%s \n", v)

			fetchContent(&cl)

			if err = fileMerge(v); err != nil {
				logrus.Error(err)
			}
		}

	}
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

	cpts := make([]string, 0)

	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

		if !info.IsDir() && strings.HasSuffix(path, ".rbx") {
			cpts = append(cpts, path)
		}
		return err
	})

	sort.Slice(cpts, func(i, j int) bool {
		ci, _ := strconv.Atoi(strings.TrimSuffix(filepath.Base(cpts[i]), ".rbx"))
		cj, _ := strconv.Atoi(strings.TrimSuffix(filepath.Base(cpts[j]), ".rbx"))
		return  ci < cj
	})

	for _, v := range cpts {

		fp, err := os.Open(v)

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

	bWriter.Flush()

	return nil
}

func sendmail(to, file, name string) {

	m := gomail.NewMessage()
	m.SetHeader("From", "guhao022@163.com")
	m.SetHeader("To", to)
	// m.SetAddressHeader("Cc", "dan@example.com", "Dan") //抄送
	m.SetHeader("Subject", "小说: "+name) // 邮件标题
	m.SetBody("text/html", name) // 邮件内容
	m.Attach(file) //附件

	d := gomail.NewDialer("smtp.163.com", 25, "guhao022@163.com", "guhao_19890412")
	d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}

func write(file string, content []byte) error {

	fpath := path.Join(file)

	basepath := path.Dir(fpath)
	// 检测文件夹是否存在   若不存在  创建文件夹
	if _, err := os.Stat(basepath); err != nil {

		if os.IsNotExist(err) {

			err = os.MkdirAll(basepath, os.ModePerm)

			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	f, err := os.OpenFile(fpath, os.O_CREATE|os.O_RDWR, os.ModePerm)

	if err != nil {
		return err
	}

	_, err = f.Write(content)

	return err
}

func DecodeGBK(s []byte) ([]byte, error) {
	reader := simplifiedchinese.GB18030.NewDecoder().Reader(bytes.NewReader(s))

	return ioutil.ReadAll(reader)
}
