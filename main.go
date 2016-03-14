package itlookslike

import (
	"appengine"
	"appengine/datastore"
	"appengine/urlfetch"
	"encoding/xml"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"golang.org/x/tools/blog/atom"
	"net/http"
	"sort"
	"strconv"
	"time"
	"math"
)

func init() {
	http.HandleFunc("/", feedHandler)
	http.HandleFunc("/update", updateHandler)
}

// 更新
func updateHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)

	now := time.Now()

	// 更新は日本時間で 10:00 - 22:00 まで
	jst := time.FixedZone("Asia/Tokyo", 9*60*60)
	nowJST := now.In(jst)
	if nowJST.Hour() < 10 || 22 < nowJST.Hour() {
		fmt.Fprintf(w, "Open: 10:00 am - Close: 10:00 pm")
		return
	}

	// xml をフェッチ
	api := fmt.Sprintf(ApiUrl, Coordinates, AppId)
	c := appengine.NewContext(r)
	client := urlfetch.Client(c)
	resp, _ := client.Get(api)
	doc, _ := goquery.NewDocumentFromResponse(resp)

	// xml をパース
	observation, forecasts := parseFeed(doc, jst)

	// 前回のデータを取得
	noData := false
	var lastResult Result
	lastResult.Status = NoRain
	lastResult.Text = ""
	key := datastore.NewKey(c, "weatherinfo", "last", 0, nil)
	errLastResult := datastore.Get(c, key, &lastResult)
	if errLastResult != nil {
		noData = true
	}

	// 結果を作成
	unixTime := now.Unix()
	result := makeResult(unixTime, &lastResult, &observation, forecasts)

	// 結果を書き込み
	_, errLastResult = datastore.Put(c, key, result)
	if errLastResult != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, errLastResult.Error())
		return
	}

	// 前回のステータスと同じならフィードを更新しない
	if !noData && lastResult.Status == result.Status {
		fmt.Fprintf(w, "skipped")
		return
	}

	// フィードを作成
	person := atom.Person{Name: FeedUri}
	feed := atom.Feed{
		Title:   "短期天気予報 (10:00 am - 10:00 pm 更新)",
		ID:      FeedUri,
		Link:    []atom.Link{atom.Link{Rel: "self", Href: FeedUri}},
		Updated: atom.Time(now),
		Author:  &person,
		Entry: []*atom.Entry{
			&atom.Entry{
				Title:   "■" + SpotName,
				ID:      FeedUri,
				Link:    []atom.Link{atom.Link{Rel: "self", Href: FeedUri}},
				Updated: atom.Time(now),
				Summary: &atom.Text{Type: "text/plain", Body: result.Text},
			}},
	}

	// フィードの書き込み
	resultKey := datastore.NewKey(c, "weatherinfo", "feed", 0, nil)
	var sd StoredData
	data, err := xml.Marshal(feed)
	if err != nil {
		fmt.Fprintf(w, err.Error())
		return
	}
	sd = StoredData{
		Serialized: []byte(xml.Header + string(data)),
	}
	datastore.Put(c, resultKey, &sd)

	return
}

// フィードをパースする
func parseFeed(doc *goquery.Document, loc *time.Location) (observation WeatherInfo, forecasts WeatherInfos) {
	forecasts = make(WeatherInfos, 0, 10)
	inputDateLayout := "200601021504"

	doc.Find("YDF Feature Property WeatherList Weather").Each(func(i int, s *goquery.Selection) {
		t := s.Find("Type").Text()
		d, _ := time.ParseInLocation(inputDateLayout, s.Find("Date").Text(), loc)
		ut := d.Unix()
		r := s.Find("Rainfall").Text()
		rf, _ := strconv.ParseFloat(r, 64)

		switch t {
		case "observation":
			// 測定値
			observation = WeatherInfo{Type: t, Time: ut, Rainfall: rf}
			break
		case "forecast":
			// 雨予報
			forecasts = append(forecasts, WeatherInfo{Type: t, Time: ut, Rainfall: rf})
			break
		}
	})

	sort.Sort(forecasts)

	return observation, forecasts
}

// 結果を作成
func makeResult(unixTime int64, lastResult *Result, observation *WeatherInfo, forecasts WeatherInfos) (result *Result) {
	// 今まで雨が降っていた
	if BeginRain < lastResult.Status && lastResult.Status < EndRain {
		// 前回からあまり時間がたっていないならまだ雨扱い
		interval := int64(60 * 30)
		if lastResult.RainTime+interval > unixTime {
			return &Result{UpdatedTime: unixTime, RainTime: lastResult.RainTime, Status: lastResult.Status, Text: lastResult.Text}
		}
		// 降り続いている
		if observation.Rainfall > RainThreshould {
			return makeStatusRain(unixTime, lastResult, observation, forecasts)
		}
	}

	// 降り始め
	if observation.Rainfall > RainThreshould {
		// 20分以内に雨予報がないなら雨扱いにしない
		rainfall, forecastTime := getRainForecast(forecasts, 1)

		if rainfall > 0 && forecastTime-unixTime <= 20*60 {
			return makeStatusRain(unixTime, lastResult, observation, forecasts)
		}
	}

	// 降ってない
	return makeStatusNoRain(unixTime, lastResult, observation, forecasts)
}

// 残り時間 timeLeft を日本語に
func getTextOfIntervalTime(timeLeft int64) string {
	if timeLeft < 10*60 {
		return "いまにも"
	}
	if timeLeft < 30*60 {
		return "まもなく"
	}
	return "もうすぐ"
}

// 複数回の予報を必要とすることで雨予報の感度を落とす
func getRainForecast(forecasts WeatherInfos, requireCount int) (rainfall float64, time int64) {
	num := 0
	for _, v := range forecasts {
		if v.Rainfall > RainThreshould {
			num++
		}
	}

	// 降りそうではない
	if num < requireCount {
		return
	}

	var t int64
	t = math.MaxInt64
	for _, v := range forecasts {
		if v.Rainfall > rainfall {
			rainfall = v.Rainfall
		}
		if v.Time < t {
			t = v.Time
		}
	}
	time = t
	return
}

// 雨ではない
func makeStatusNoRain(unixTime int64, lastResult *Result, observation *WeatherInfo, forecasts WeatherInfos) (result *Result) {
	rainfall, forecastTime := getRainForecast(forecasts, 2)

	var text string
	// 雨が降りそう
	if rainfall > 0 {
		rainStatus := getForecastRainStatus(rainfall)
		text = getTextOfIntervalTime(forecastTime-unixTime) + " " + rainStatusAsText[rainStatus] + " が降りそうです"
		return &Result{UpdatedTime: unixTime, RainTime: 0, Status: rainStatus, Text: text}
	}

	// 今までは雨が降っていた
	if BeginRain < lastResult.Status && lastResult.Status < EndRain {
		text = "もう雨は上がりました"
		return &Result{UpdatedTime: unixTime, RainTime: 0, Status: NoRain, Text: text}
	}

	// 降っていないし予報もない
	text = "現在、雨は降っていません"
	return &Result{UpdatedTime: unixTime, RainTime: 0, Status: NoRain, Text: text}
}

// 雨が降っている
func makeStatusRain(unixTime int64, lastResult *Result, observation *WeatherInfo, forecasts WeatherInfos) (result *Result) {
	rainStatus := getRainStatus(observation.Rainfall)
	text := "現在 " + rainStatusAsText[rainStatus] + " が降っています"
	return &Result{UpdatedTime: unixTime, RainTime: unixTime, Status: rainStatus, Text: text}
}

// フィードを返す
func feedHandler(w http.ResponseWriter, r *http.Request) {
	c := appengine.NewContext(r)

	var sd StoredData

	key := datastore.NewKey(c, "weatherinfo", "feed", 0, nil)
	datastore.Get(c, key, &sd)

	w.Header().Set("Content-Type", "application/atom+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(sd.Serialized)
}
