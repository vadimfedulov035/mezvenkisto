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

// program init config
const InitJSON = "/etc/mezvenkisto/token.json"
// time period configs
const DayJSON = "data_day.json"
const MonthJSON = "data_month.json"
const YearJSON = "data_year.json"
// maximum number of speakers
const SpeakersMaxNum = 10
// time period log messages
const NewDayLog = "New day %02d/%02d/%02d has come.\n"
const NewMonthLog = "New month %02d/%02d has come.\n"
const NewYearLog = "New year %02d has come.\n"
const RenewalLog = "Renewal.\n"
// hashtag
const Hashtag = "#mezvenko"
// basic date info to show
const DateD = "%02d/%02d/%02d:"
const DateM = "%02d/%02d:"
const DateY = "%02d:"
// compact durations
const Second = time.Second
const FiveSeconds = 5 * time.Second
const Minute = 60 * time.Second
// summary parts
const SummaryHr = "%02dh %02dm %02ds"
const SummaryMin = "%02dm %02ds"
const SummarySec = "%02ds"
const Place = "%d. %s:\n%s\n"
// video
const Video1H = "https://www.youtube.com/watch?v=Zt_DIhbYNa4"
const Video2H = "https://www.youtube.com/watch?v=4ilbLTR5rg8"
const Video27H = "https://www.youtube.com/watch?v=sijVvMsNCNo"

type InitConfig struct {
	KeyAPI string
	ChatID int64
}

type Update struct {
	Name string
	DurationU uint32
}

type ConfigData struct {
	Durations map[string]uint32 `json:"durations"`
	Duration uint32             `json:"duration"`
}

type DurationData struct {
	Name     string
	Duration uint32
}

type DateInfo struct {
	Year      int
	Month     int
	MonthName string
	Day       int
}

func loadInitConfig(config string) (string, int64) {
	var initConfig InitConfig
	data, err := os.ReadFile(config)
	if err != nil {
		panic(err)
	} else {
		json.Unmarshal(data, &initConfig)
	}
	return initConfig.KeyAPI, initConfig.ChatID
}

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

func getUpdate(msg *tgbotapi.Message) Update {
	// get name
	name := getName(msg)
	// get duration
	d := 0
	if msg.Voice != nil {
		d = msg.Voice.Duration
	} else if msg.VideoNote != nil {
		d = msg.VideoNote.Duration
	} else if msg.Audio != nil {
		if msg.Caption != "" && strings.Contains(msg.Caption, Hashtag) {
			d = msg.Audio.Duration
		}
	} else if msg.Video != nil {
		if msg.Caption != "" && strings.Contains(msg.Caption, Hashtag) {
			d = msg.Video.Duration
		}
	}
	// return
	update := Update{
		Name: name,
		DurationU: uint32(d),
	}
	return update
}

// renew duration(s)
func renew(update Update, durations map[string]uint32, duration *uint32) {
	var info string
	// get name and duration update
	name := update.Name
	d := update.DurationU
	// duration
	durationOld := *duration
	*duration += d
	info = fmt.Sprintf("%d+%d=%d", durationOld, d, *duration)
	// durations
	if name != "" {
		dPersonOld := durations[name]
		durations[name] += d
		dPerson := durations[name]
		info += fmt.Sprintf(" from %s", name)
		info += fmt.Sprintf("[%d+%d=%d]", dPersonOld, d, dPerson)
	}
	// log renewal end
	log.Println(info)
}

func calcTime(duration uint32) [3]int {
	durationTime := time.Duration(duration) * time.Second
	hours := int(durationTime.Hours())
	minutes := int(durationTime.Minutes()) % 60
	seconds := int(durationTime.Seconds()) % 60
	time := [3]int{hours, minutes, seconds}
	return time
}

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

func getSummary(duration uint32) string {
	var summary string
	time := calcTime(duration)
	hours, minutes, seconds := time[0], time[1], time[2]
	if hours != 0 {
		summary = fmt.Sprintf(SummaryHr, hours, minutes, seconds)
	} else if minutes != 0 {
		summary = fmt.Sprintf(SummaryMin, minutes, seconds)
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
	// sort based on duration
	durationData := make([]DurationData, speakersNum)
	for k, v := range durations {
		durationDataU := DurationData{Name: k, Duration: v}
		durationData = append(durationData, durationDataU)
	}
	sort.Slice(durationData, func(i, j int) bool {
		return durationData[i].Duration > durationData[j].Duration
	})
	// convert to human readable format
	for i := 0; i < speakersNum; i++ {
		name := durationData[i].Name
		summaryPart = getSummary(durationData[i].Duration)
		summaryPersonal += fmt.Sprintf(Place, i+1, name, summaryPart)
	}
	return summaryPersonal
}

func summarise(config string, date string) string {
	durations, duration := loadConfig(config)
	summaryT := getSummary(duration)
	summaryP := getSummaryPersonal(durations)
	video := getVideo(duration)
	summary := date + " " + summaryT + "\n\n" + summaryP + "\n" + video
	return summary
}

func loadConfig(config string) (map[string]uint32, uint32) {
	var configData ConfigData
	var durations map[string]uint32
	var duration uint32
	var info string
	// load config data
	file, _ := os.OpenFile(config, os.O_RDWR|os.O_CREATE, 0644)
	file.Close()
	jsonData, _ := os.ReadFile(config)
	json.Unmarshal(jsonData, &configData)
	// load durations
	if configData.Durations != nil {
		durations = configData.Durations
		duration = configData.Duration
		info = fmt.Sprintf("%s -> memory\n", config)
	} else {
		durations = map[string]uint32{}
		duration = 0
		info = fmt.Sprintf("0 -> %s -> memory\n", config)
	}
	// load duration
	duration = configData.Duration
	// log loading end
	log.Println(info)
	return durations, duration
}

func saveConfig(config string, durations map[string]uint32, duration uint32) {
	// write renewed config data to config
	configData := ConfigData{
		Durations: durations,
		Duration: duration,
	}
	jsonData, _ := json.Marshal(configData)
	os.WriteFile(config, jsonData, 0644)
}

func updateConfig(config string, configU string) {
	// log update start and end
	defer log.Printf("%s -> %s [✓]\n", config, configU)
	log.Printf("%s -> %s […]\n", config, configU)
	// get new and old config data
	durations, duration := loadConfig(config)
	durationsOld, durationOld := loadConfig(configU)
	// update old config data
	for k, v := range durations {
		durationsOld[k] += v
	}
	durationOld += duration
	// save updated old data
	saveConfig(configU, durationsOld, durationOld)
}

func resetConfig(config string) {
	// log reset end
	defer log.Printf("%s <- 0\n", config)
	// set empty variables
	durations := make(map[string]uint32)
	var duration uint32 = 0
	// reset config
	saveConfig(config, durations, duration)
}

func main() {
	// initialize bot
	KeyAPI, ChatID := loadInitConfig(InitJSON)
	bot, err := tgbotapi.NewBotAPI(KeyAPI)
	if err != nil {
		log.Panic(err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)
	// initalize all data variables
	configs := map[string]string{
		"day": DayJSON,
		"month": MonthJSON,
		"year": YearJSON,
	}
	// initialize mutex
	var mutex sync.Mutex
	// initalize channel to reload config
	reload := make(chan bool, 1)
	// every 60 seconds check for new message
	go func(reload <-chan bool) {
		// load previous day data
		config := configs["day"]
		mutex.Lock()
		durations, duration := loadConfig(config)
		mutex.Unlock()
		for update := range updates {
			// reload day config on signal
			select {
			case <- reload:
				mutex.Lock()
				durations, duration = loadConfig(config)
				mutex.Unlock()
			// every message update duration and save day config
			default:
				msg := update.Message
				if msg != nil {
					u := getUpdate(msg)
					if u.DurationU > 0 {
						renew(u, durations, &duration)
					}
					mutex.Lock()
					saveConfig(config, durations, duration)
					mutex.Unlock()
				}
			}
		}
	}(reload)
	// every second check for period passing
	go func(reload chan<- bool) {
		now := time.Now()
		pDay, pMonth, pYear := now.Day(), int(now.Month()), now.Year()
		for {
			newDate := false
			var summary [3]string
			var period, periodU string
			var config, configU string
			var date string
			// get date info
			now = time.Now()
			day := now.Day()
			month := int(now.Month())
			year := now.Year()
			// every day update month config, summarize and reset
			if day != pDay {
				newDate = true
				period, periodU = "day", "month"
				config = configs[period]
				configU = configs[periodU]
				log.Printf(NewDayLog, day, month, year)
				date = fmt.Sprintf(DateD, pDay, pMonth, pYear)
				summary[0] = summarise(config, date)
				mutex.Lock()
				updateConfig(config, configU)
				resetConfig(configs[period])
				mutex.Unlock()
			}
			// every month update year config, summarize and reset
			if month != pMonth {
				newDate = true
				period, periodU = "month", "year"
				config = configs[period]
				configU = configs[periodU]
				log.Printf(NewMonthLog, month, year)
				date = fmt.Sprintf(DateM, pMonth, pYear)
				summary[1] = summarise(config, date)
				mutex.Lock()
				updateConfig(config, configU)
				resetConfig(config)
				mutex.Unlock()
			}
			// every year summarize and reset year config
			if year != pYear {
				newDate = true
				period = "year"
				config = configs[period]
				log.Printf(NewYearLog, year)
				date = fmt.Sprintf(DateY, pYear)
				summary[2] = summarise(config, date)
				mutex.Lock()
				resetConfig(config)
				mutex.Unlock()
			}
			// every new date log, message, signal and sleep
			if newDate {
				// log renewal
				log.Printf(RenewalLog)
				// send summary
				for _, s := range summary {
					if s == "" {
						break
					}
					msg := tgbotapi.NewMessage(ChatID, s)
					for {
						_, err := bot.Send(msg)
						if err != nil {
							log.Println(err)
							time.Sleep(FiveSeconds)
						} else {
							break
						}
					}
					time.Sleep(Second)
				}
				// send renewal signal
				reload <- true
				time.Sleep(Minute)
			}
			// record date info
			pDay, pMonth, pYear = day, month, year
			time.Sleep(Second)
		}
	}(reload)
	select {}
}
