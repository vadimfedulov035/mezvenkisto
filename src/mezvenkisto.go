package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// program init conf
const InitJSON = "/etc/mezvenkisto/token.json"

// time period confs
const DayJSON = "data_day.json"
const MonthJSON = "data_month.json"
const YearJSON = "data_year.json"
const TotalJSON = "data_total.json"

// maximum number of speakers
const SpeakersMaxNum = 25
const ReloadLog = "Reload.\n"

// hashtag
const Hashtag = "#mezvenko"

// basic date info to show
const DayDate = "%02d/%02d/%02d:"
const MonthDate = "%02d/%02d:"
const YearDate = "%02d:"

// summary parts
const SummaryHr = "%02dh %02dm %02ds"
const SummaryMin = "%02dm %02ds"
const SummarySec = "%02ds"
const Place = "%d. %s:\n%s\n"

// video
const Video1H = "https://www.youtube.com/watch?v=Zt_DIhbYNa4"
const Video2H = "https://www.youtube.com/watch?v=4ilbLTR5rg8"
const Video27H = "https://www.youtube.com/watch?v=sijVvMsNCNo"

type InitConf struct {
	KeyAPI string
	ChatID int64
}

type Upd struct {
	Name      string
	DurationU uint32
}

type ConfData struct {
	Durations map[string]uint32 `json:"durations"`
	Duration  uint32            `json:"duration"`
}

type DurationData struct {
	Name     string
	Duration uint32
}

//                      ____ ___  _   _ _____ ___ ____                      //
//                     / ___/ _ \| \ | |  ___|_ _/ ___|                     //
//                    | |  | | | |  \| | |_   | | |  _                      //
//                    | |__| |_| | |\  |  _|  | | |_| |                     //
//                     \____\___/|_| \_|_|   |___\____|                     //

func loadInitConf(conf string) (string, int64) {
	var initConf InitConf
	data, err := os.ReadFile(conf)
	if err != nil {
		panic(err)
	} else {
		json.Unmarshal(data, &initConf)
	}
	return initConf.KeyAPI, initConf.ChatID
}

func loadConf(conf string) (map[string]uint32, uint32) {
	var confData ConfData
	var durations map[string]uint32
	var duration uint32
	// load config data
	file, _ := os.OpenFile(conf, os.O_RDWR|os.O_CREATE, 0644)
	file.Close()
	jsonData, _ := os.ReadFile(conf)
	json.Unmarshal(jsonData, &confData)
	// load durations
	if confData.Durations != nil {
		durations = confData.Durations
		duration = confData.Duration
	} else {
		durations = map[string]uint32{}
		duration = 0
	}
	// load duration
	duration = confData.Duration
	return durations, duration
}

func saveConf(conf string, durations map[string]uint32, duration uint32) {
	confData := ConfData{
		Durations: durations,
		Duration:  duration,
	}
	jsonData, _ := json.Marshal(confData)
	os.WriteFile(conf, jsonData, 0644)
}

func resetConf(conf string) {
	defer fmt.Printf("%s <- 0\n", conf)
	// set data to zero
	durations := make(map[string]uint32)
	var duration uint32 = 0
	// save with zero-data
	saveConf(conf, durations, duration)
}

func updateConf(conf string, confU string) {
	// get new and old conf data
	durations, duration := loadConf(conf)
	durationsOld, durationOld := loadConf(confU)
	// update old conf data (increment)
	for k, v := range durations {
		durationsOld[k] += v
	}
	durationOld += duration
	// save updated data
	saveConf(confU, durationsOld, durationOld)
}

// update configs related to period
func updatePeriod(period string) {
	// get period to update
	periodsU := map[string]string{
		"day":   "month",
		"month": "year",
		"year":  "total",
	}
	periodU := periodsU[period]
	// get config to reset and config to update
	confs := map[string]string{
		"day":   DayJSON,
		"month": MonthJSON,
		"year":  YearJSON,
		"total": TotalJSON,
	}
	conf, confU := confs[period], confs[periodU]
	// log update
	fmt.Printf("%s update […]\n", period)
	defer fmt.Printf("%s -> %s [✓]\n", conf, confU)
	defer fmt.Printf("%s update [✓]\n", period)
	// update and reset configs
	updateConf(conf, confU)
	resetConf(conf)
}

//                    _   _ ____  ____    _  _____ _____                    //
//                   | | | |  _ \|  _ \  / \|_   _| ____|                   //
//                   | | | | |_) | | | |/ _ \ | | |  _|                     //
//                   | |_| |  __/| |_| / ___ \| | | |___                    //
//                    \___/|_|   |____/_/   \_\_| |_____|                   //

func getName(msg *tgbotapi.Message) string {
	var name string
	userName := msg.From.UserName
	firstName := msg.From.FirstName
	// username
	if userName != "" {
		name = "@" + userName
		// first name
	} else if firstName != "" {
		name = firstName
	}
	return name
}

func getUpdate(msg *tgbotapi.Message) Upd {
	var upd Upd
	name := getName(msg)
	durationU := 0
	if msg.Voice != nil {
		durationU = msg.Voice.Duration
	} else if msg.VideoNote != nil {
		durationU = msg.VideoNote.Duration
	} else if msg.Audio != nil {
		if msg.Caption != "" && strings.Contains(msg.Caption, Hashtag) {
			durationU = msg.Audio.Duration
		}
	} else if msg.Video != nil {
		if msg.Caption != "" && strings.Contains(msg.Caption, Hashtag) {
			durationU = msg.Video.Duration
		}
	}
	upd.Name = name
	upd.DurationU = uint32(durationU)
	return upd
}

func updateDuration(upd Upd, durations map[string]uint32, duration *uint32) {
	var info string
	// get name and duration update
	name := upd.Name
	durationU := upd.DurationU
	// update duration and log
	durationOld := *duration
	*duration += durationU
	info = fmt.Sprintf("%d+%d=%d", durationOld, durationU, *duration)
	// update durations and log
	if name != "" {
		dPersonOld := durations[name]
		durations[name] += durationU
		dPerson := durations[name]
		info += fmt.Sprintf(" from %s", name)
		info += fmt.Sprintf("[%d+%d=%d]", dPersonOld, durationU, dPerson)
	}
	fmt.Println(info)
}

//         ____  _   _ __  __ __  __    _    ____  ___ __________           //
//        / ___|| | | |  \/  |  \/  |  / \  |  _ \|_ _|__  / ____|          //
//        \___ \| | | | |\/| | |\/| | / _ \ | |_) || |  / /|  _|            //
//         ___) | |_| | |  | | |  | |/ ___ \|  _ < | | / /_| |___           //
//        |____/ \___/|_|  |_|_|  |_/_/   \_\_| \_\___/____|_____|          //

// used to get general summary and personal summary
func getSummary(duration uint32) string {
	var summary string
	time := calcTime(duration)
	hours, minutes, seconds := time[0], time[1], time[2]
	// for hours 00h 00m 00s
	if hours != 0 {
		summary = fmt.Sprintf(SummaryHr, hours, minutes, seconds)
		// for minutes 00m 00s
	} else if minutes != 0 {
		summary = fmt.Sprintf(SummaryMin, minutes, seconds)
		// for seconds 00s
	} else {
		summary = fmt.Sprintf(SummarySec, seconds)
	}
	return summary
}

func getSummaryPersonal(durations map[string]uint32) string {
	var summaryPart, summaryPersonal string
	// limit to maximum number of speakers
	speakersNum := SpeakersMaxNum
	if len(durations) < SpeakersMaxNum {
		speakersNum = len(durations)
	}
	// sort results based on duration
	durationData := make([]DurationData, speakersNum)
	for k, v := range durations {
		durationDataU := DurationData{Name: k, Duration: v}
		durationData = append(durationData, durationDataU)
	}
	sort.Slice(durationData, func(i, j int) bool {
		return durationData[i].Duration > durationData[j].Duration
	})
	// convert results to human readable format
	for i := 0; i < speakersNum; i++ {
		name := durationData[i].Name
		summaryPart = getSummary(durationData[i].Duration)
		summaryPersonal += fmt.Sprintf(Place, i+1, name, summaryPart)
	}
	return summaryPersonal
}

// convert duration to hours, minutes, seconds
func calcTime(duration uint32) [3]int {
	durationTime := time.Duration(duration) * time.Second
	hours := int(durationTime.Hours())
	minutes := int(durationTime.Minutes()) % 60
	seconds := int(durationTime.Seconds()) % 60
	time := [3]int{hours, minutes, seconds}
	return time
}

// decide what video to show (if any)
func getVideo(duration uint32) string {
	var video string
	time := calcTime(duration)
	hours := time[0]
	if hours >= 27 {
		video = Video27H
	} else if hours >= 2 {
		video = Video2H
	} else if hours >= 1 {
		video = Video1H
	}
	return video
}

func getDate(period string, t time.Time) string {
	var date string
	day, month, year := t.Day(), int(t.Month()), t.Year()
	// print date according to period
	switch period {
	case "day":
		date = fmt.Sprintf(DayDate, day, month, year)
	case "month":
		date = fmt.Sprintf(MonthDate, month, year)
	case "year":
		date = fmt.Sprintf(YearDate, year)
	}
	return date
}

// get complete summary
func summarize(period string, t time.Time) string {
	// get config to load
	confs := map[string]string{
		"day":   DayJSON,
		"month": MonthJSON,
		"year":  YearJSON,
		"total": TotalJSON,
	}
	conf := confs[period]
	// load durations from config
	durations, duration := loadConf(conf)
	// make durations human readable summaries
	summaryT := getSummary(duration)
	summaryP := getSummaryPersonal(durations)
	// get video to show (if any)
	video := getVideo(duration)
	// get date to show
	date := getDate(period, t)
	summary := date + " " + summaryT + "\n\n" + summaryP + "\n" + video
	return summary
}

//         ____  ___  ____   ___  _   _ _____ ___ _   _ _____ ____          //
//        / ___|/ _ \|  _ \ / _ \| | | |_   _|_ _| \ | | ____/ ___|         //
//       | |  _| | | | |_) | | | | | | | | |  | ||  \| |  _| \___ \         //
//       | |_| | |_| |  _ <| |_| | |_| | | |  | || |\  | |___ ___) |        //
//        \____|\___/|_| \_\\___/ \___/  |_| |___|_| \_|_____|____/         //

func handleDurationU(bot *tgbotapi.BotAPI, mutex *sync.Mutex, r <-chan bool) {
	// load data or init if none
	conf := DayJSON
	mutex.Lock()
	durations, duration := loadConf(conf)
	mutex.Unlock()

	// set bot
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for {
		select {
		// every signal reload day config
		case <-r:
			fmt.Printf(ReloadLog)
			mutex.Lock()
			durations, duration = loadConf(conf)
			mutex.Unlock()

			// every message update duration
		case update := <-updates:
			msg := update.Message
			if msg != nil {
				upd := getUpdate(msg)
				if upd.DurationU > 0 {
					updateDuration(upd, durations, &duration)
					mutex.Lock()
					saveConf(conf, durations, duration)
					mutex.Unlock()
				}
			}
		}
	}
}

func handlePeriodU(mutex *sync.Mutex, r chan<- bool, s chan<- string) {
	summaries := [3]string{"", "", ""}

	// wait until midnight
	now := time.Now()
	nextDay := now.AddDate(0, 0, 1)
	nYear, nMonth, nDay := nextDay.Year(), nextDay.Month(), nextDay.Day()
	midnight := time.Date(nYear, nMonth, nDay, 0, 0, 0, 0, now.Location())
	timer := time.NewTimer(time.Until(midnight))
	<-timer.C

	// set ticker for every day
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// every day summarize, update and reset configs
	for ; true; <-ticker.C {
		mutex.Lock()

		// new day
		now = time.Now()
		then := now.AddDate(0, 0, -1)
		summaries[0] = summarize("day", then)
		updatePeriod("day")

		// new month
		firstDay := (now.Day() == 1)
		if firstDay {
			summaries[1] = summarize("month", then)
			updatePeriod("month")
		}

		// new year
		firstMonth := (int(now.Month()) == 1)
		if firstMonth && firstDay {
			summaries[2] = summarize("year", then)
			updatePeriod("year")
		}

		// send summary
		for _, summary := range summaries {
			if summary != "" {
				s <- summary
			}
		}

		mutex.Unlock()

		// reload config
		r <- true
	}
}

func handleSummary(bot *tgbotapi.BotAPI, ChatID int64, s <-chan string) {
	for {
		select {
		case summary := <-s:
			msg := tgbotapi.NewMessage(ChatID, summary)
			for {
				_, err := bot.Send(msg)
				if err != nil {
					fmt.Println(err)
					time.Sleep(5 * time.Second)
				} else {
					break
				}
			}
			time.Sleep(time.Second)
		}
	}
}

func main() {
	// initialize bot
	KeyAPI, ChatID := loadInitConf(InitJSON)
	bot, err := tgbotapi.NewBotAPI(KeyAPI)
	if err != nil {
		log.Panic(err)
	}

	// mutex
	var mutex sync.Mutex

	// reload channel
	r := make(chan bool, 1)

	// summary channel
	s := make(chan string, 3)

	// start goroutines
	go handleDurationU(bot, &mutex, r)
	go handlePeriodU(&mutex, r, s)
	go handleSummary(bot, ChatID, s)
	select {}
}
