package itlookslike

type RainStatus int

const (
	NoRain RainStatus = iota // 降っていない

	BeginRain
	LightRain    // 小雨 数時間続いても1mm未満の雨
	SoftRain     // 弱い雨 3mm未満
	ModerateRain // やや強い雨 10mm以上20mm未満
	HeavyRain    // 強い雨 20mm以上
	EndRain

	BeginForecastRain
	ForecastSoftRain     // 弱い雨 3mm未満
	ForecastModerateRain // やや強い雨 10mm以上20mm未満
	ForecastHeavyRain    // 強い雨 20mm以上
	EndForecastRain
)

// 雨量から RainStatus を返す
func getRainStatus(rainfall float64) RainStatus {
	if 0 == rainfall {
		return NoRain
	}
	if 1 > rainfall {
		return LightRain
	}
	if 3 > rainfall {
		return SoftRain
	}
	if 20 > rainfall {
		return ModerateRain
	}
	return HeavyRain
}

// 予報雨量から ForecastRainStatus を返す
func getForecastRainStatus(rainfall float64) RainStatus {
	if 0 == rainfall {
		return NoRain
	}
	if 3 > rainfall {
		return ForecastSoftRain
	}
	if 20 > rainfall {
		return ForecastModerateRain
	}
	return ForecastHeavyRain
}

// RainStatus から文字列を返す
var rainStatusAsText = map[RainStatus]string{
	NoRain:               "雨は降っていません",
	LightRain:            "小雨",
	SoftRain:             "弱い雨",
	ModerateRain:         "やや強い雨",
	HeavyRain:            "強い雨",
	ForecastSoftRain:     "弱い雨",
	ForecastModerateRain: "やや強い雨",
	ForecastHeavyRain:    "強い雨",
}

// 取得した気象情報
type WeatherInfo struct {
	Type     string  // observation or forecast
	Time     int64   // Unix time
	Rainfall float64 // mm/h
}

// 予報時刻でのソート用
type WeatherInfos []WeatherInfo

func (w WeatherInfos) Len() int {
	return len(w)
}
func (w WeatherInfos) Swap(i, j int) {
	w[i], w[j] = w[j], w[i]
}
func (w WeatherInfos) Less(i, j int) bool {
	return w[i].Time < w[j].Time
}

// 前回の情報
type Result struct {
	UpdatedTime int64 // 前回の更新時刻 Unix time
	RainTime    int64 // 最後に実際に雨が降っていた時刻 Unix time
	Status      RainStatus
	Text        string
}

// フィードの出力用
type StoredData struct {
	Serialized []byte
}
