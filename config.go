package itlookslike

const ApiUrl = "http://weather.olp.yahooapis.jp/v1/place?coordinates=%s&appid=%s"

// 雨と判断する下限値 mm/h
const RainThreshould = 0.5

// 予報位置。表示用
const SpotName = "新宿"
const FeedUri = "http://weather.yahoo.co.jp/weather/jp/13/4410/13104.html"

// 予報位置。api のパラメーター。デフォルトでは都庁のあたりです
const Coordinates = "139.691764,35.689661"

// http://developer.yahoo.co.jp/ で取得したあなたのアプリケーション ID
const AppId = "あなたのアプリケーション ID"
