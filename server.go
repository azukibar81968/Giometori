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
	///script
	var startMessage_begin = "スタート！最初の言葉は"
	var startMessage_end = "だよ！"
	var announceMessage_begin = "次は「"
	var announceMessage_end = "」からスタートだよ！"
	var nonLocateErrMessage string = "それは地名じゃない見たい...!"
	var shiritoriErrMessage string = "しりとりになってないよ！！"
	var imageErrMessage string = "の画像が見つからなかった！ごめん！"
	var initErrMessage string = "ごめん！その単語は使えない見たい...他の単語で試してみて！"

	var prev_lastText string = "__initText"

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
					log.Print("//////" + "revieve message rawMessage=" + message.Text + "///////")
					//analyze message
					var rawMessage = message.Text
					var pronunciation string = getPronunce(m, rawMessage)
					var isLocate bool = isLocationName(m, rawMessage)

					//get first and last literature
					var this_headText string = getHeadText(pronunciation)
					var this_lastText string = getLastText(pronunciation)

					//get Image
					var imageURL string = GetImageFromKeyword(rawMessage)

					//replying
					if prev_lastText == "__initText" { //////最初のひとこと目

						if this_lastText != "" {
							prev_lastText = this_lastText
							// reply testMessage
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewImageMessage(imageURL, imageURL), linebot.NewTextMessage(startMessage_begin+rawMessage+"("+pronunciation+")"+startMessage_end)).Do(); err != nil {
								log.Print(err)
							}
						} else if isLocate == false {
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(nonLocateErrMessage)).Do(); err != nil {
								log.Print(err)
							}
						} else {
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(initErrMessage)).Do(); err != nil {
								log.Print(err)
							}
						}

					} else if imageURL != "" { //////普通の返信
						if prev_lastText == this_headText { //////正しいしりとり
							prev_lastText = this_lastText
							// reply Picture
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewImageMessage(imageURL, imageURL), linebot.NewTextMessage(announceMessage_begin+prev_lastText+announceMessage_end)).Do(); err != nil {
								log.Print(err)
							}
						} else if isLocate == false { //////地名じゃない
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(nonLocateErrMessage), linebot.NewTextMessage(announceMessage_begin+prev_lastText+announceMessage_end)).Do(); err != nil {
								log.Print(err)
							}
						} else { //////正しくないしりとり
							if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(shiritoriErrMessage), linebot.NewTextMessage(announceMessage_begin+prev_lastText+announceMessage_end)).Do(); err != nil {
								log.Print(err)
							}
						}
					} else { //////画像が取得できなかった場合
						if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(rawMessage+imageErrMessage), linebot.NewTextMessage(announceMessage_begin+prev_lastText+announceMessage_end)).Do(); err != nil {
							log.Print(err)
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

func getPronunce(m *mecab.MeCab, word string) string {
	log.Print("getPronunce....")
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
	var ans string = ""

	for {
		features := strings.Split(node.Feature(), ",")
		log.Print("形態素解析:" + node.Feature())
		if features[0] == "名詞" && len(features) >= 8 {
			ans += features[7]
		}
		if node.Next() != nil {
			break
		}
	}
	log.Print("get pronunce:" + ans)
	return ans
}

func getLastText(w string) string {
	var ans string
	r_w := []rune(w)
	for cnt := 1; cnt < len(r_w); cnt++ {
		if len(r_w) != 0 {
			ans = string(r_w[len(r_w)-cnt:])
		} else {
			ans = "err"
		}

		if ans != "ー" && ans != "ッ" {
			break
		}
	}
	return ans
}
func getHeadText(w string) string {
	var ans string
	r_w := []rune(w)

	if len(r_w) != 0 {
		ans = string(r_w[:1])
	} else {
		ans = "err"
	}
	return ans
}

func isLocationName(m *mecab.MeCab, word string) bool {
	log.Print("check isLocate....")
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
	var ans bool = false

	for {
		features := strings.Split(node.Feature(), ",")
		//log.Print("形態素解析:" + node.Feature())
		log.Print(features)
		if features[0] == "名詞" {
			log.Print(features[2])
			ans = features[2] == "地域"
			break
		}
		if node.Next() != nil {
			ans = false
			break
		}
	}
	if ans {
		log.Print("isLocate?: true")
	} else {
		log.Print("isLocate?: false")
	}
	return ans
}

func GetImageFromKeyword(keyword string) string {
	log.Print("searching : " + keyword + "....")
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
	search.Q(keyword + "の風景")
	search.ImgType("photo")
	search.Num(1)
	search.Start(1)
	call, err := search.Do()
	if err != nil {
		log.Fatal(err)
	}

	var result string

	for _, r := range call.Items {
		result = r.Link
		log.Print(result)
	}

	return result
}
