// Copyright 2016 LINE Corporation
//
// LINE Corporation licenses this file to you under the Apache License,
// version 2.0 (the "License"); you may not use this file except in compliance
// with the License. You may obtain a copy of the License at:
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package main

import (
	//"fmt"
	"github.com/bluele/mecab-golang"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/customsearch/v1"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

func main() {
	//init
	bot, err := linebot.New(
		"71c215ce54be8a6b261839243e5a406c",
		"8gk2FJpQ1Ma/lKSU8bv0i3FV9Z35TaEs5xXS8hpXEYYyJ8XJIdraSkOGneL/6sG8gYhtuJKtfykyi6okGsS8kiG19fs5Z2reeQBuGNZV0UWXWEAD3wYoox7Gh60BIaB3w84C+9i04qvGp/UCjqguvAdB04t89/1O/w1cDnyilFU=",
	)
	if err != nil {
		log.Fatal(err)
	}
	var prevText string = "__initText"
	var errMessage string = "しりとりになってないよ！！"
	m, err := mecab.New("-Owakati")
	if err != nil {
		panic(err)
	}
	defer m.Destroy()

	// Setup HTTP Server for receiving requests from LINE platform
	http.HandleFunc("/callback", func(w http.ResponseWriter, req *http.Request) {
		events, err := bot.ParseRequest(req)
		if err != nil {
			if err == linebot.ErrInvalidSignature {
				w.WriteHeader(400)
			} else {
				w.WriteHeader(500)
			}
			return
		}

		for _, event := range events {
			if event.Type == linebot.EventTypeMessage {
				switch message := event.Message.(type) {
				case *linebot.TextMessage:
					log.Print("//////" + "revieve message prevText=" + prevText + "///////")
					var rawMessage = message.Text
					var pronunciation string = parseToNode(m, rawMessage)
					var imageURL string = GetImageFromKeyword(rawMessage)

					log.Print("recieve:" + rawMessage)
					if prevText == "__initText" {

						prevText = pronunciation
						// reply testMessage
						if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("testMSG_first_reply"+pronunciation)).Do(); err != nil {
							log.Print(err)
						}

					} else {
						r_pronunciation := []rune(pronunciation)
						r_prevText := []rune(prevText)
						var firstText string = string(r_pronunciation[:1])
						var lastText string = string(r_prevText[len(r_prevText)-1:])

						log.Print("lastText:" + lastText)
						log.Print("firstText:" + firstText)
						if lastText == firstText {
							prevText = pronunciation
							// reply Picture
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewImageMessage(imageURL, imageURL)).Do(); err != nil {
								log.Print(err)
							}
						} else {
							// reply errMessage
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(errMessage+"_recieve:"+rawMessage+"("+pronunciation+")")).Do(); err != nil {
								log.Print(err)
							}
						}
					}
				}
			}
		}
	})
	// This is just sample code.
	// For actual use, you must support HTTPS by using `ListenAndServeTLS`, a reverse proxy or something else.
	if err := http.ListenAndServe(":8081", nil); err != nil {
		log.Fatal(err)
	}
}

func parseToNode(m *mecab.MeCab, word string) string {
	//init
	tg, err := m.NewTagger()
	if err != nil {
		panic(err)
	}
	defer tg.Destroy()
	lt, err := m.NewLattice(word)
	if err != nil {
		log.Panic(err)
	}
	defer lt.Destroy()

	//parse
	node := tg.ParseToNode(lt)

	//return
	var ans string

	for {
		features := strings.Split(node.Feature(), ",")
		if features[0] == "名詞" {
			ans = features[7]
			break
		}
		if node.Next() != nil {
			ans = ""
			break
		}
	}
	return ans
}

func GetImageFromKeyword(keyword string) string {
	data, err := ioutil.ReadFile("search-key.json")
	if err != nil {
		log.Fatal(err)
	}

	conf, err := google.JWTConfigFromJSON(data, "https://www.googleapis.com/auth/cse")
	if err != nil {
		log.Fatal(err)
	}

	client := conf.Client(oauth2.NoContext)
	cseService, err := customsearch.New(client)
	search := cseService.Cse.List()

	// 検索エンジンIDを適宜設定
	search.Cx("938fd93a468ac8dfb")
	// Custom Search Engineで「画像検索」をオンにする
	search.SearchType("image")
	search.ExactTerms(keyword + "の風景")
	search.Num(1)
	search.Start(1)
	call, err := search.Do()
	if err != nil {
		log.Fatal(err)
	}

	var result string

	for _, r := range call.Items {
		result = r.Link
	}

	return result
}
