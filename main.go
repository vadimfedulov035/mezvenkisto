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
	ifc "github.com/vadimfedulov035/ifc"
)

// config filenames fur various time periods
const DayJSON = "data_day.json"
const MonthJSON = "data_month.json"
const YearJSON = "data_year.json"
// maximum number of speakers to show info about
const SpeakersMaxNum = 10
// number of update log messages
const UpdateLogNum = 4
// time period log messages
const NewDayLog = "New day %d/%d/%d has come\n"
const NewMonthLog = "New month %d/%d has come\n"
const NewYearLog = "New year %d has come\n"
// start of summary
const Hashtag = "#mezvenko\n"
// basic date info and summary to show
const TimeExplanation = " laŭ IFK kaj UTC\n\n"
const PreviousDaySummary = "La %d de %s, %d resumita"
const PreviousMonthSummary = "%s de %d resumita"
const PreviousYearSummary = "%d resumita"
// result info
const BadResult = "TALPA VENKO OKAZAS!\nTALPOJ VENKAS!\nMalpli ol 1 horo tage!\n\n"
const GoodResult = "MEZA VENKO OKAZAS!\nNI VENKAS!\nPli ol 1 horo tage!\n\n"
const BadVideo = "https://youtu.be/Bb3ybZqK3xo"
const GoodVideoOne = "https://youtu.be/Zt_DIhbYNa4"
const GoodVideoTwo = "https://youtu.be/4ilbLTR5rg8"
const GoodVideoThree = "https://youtu.be/sijVvMsNCNo"
// hour number to achieve good result
const PassingHourBarDay = 1
const PassingHourBarMonth = 28
const PassingHourBarYear = 365
// hour number advice to achieve good result
const AdvicePassingHourBarDay = "Ni parolu pli ol 1 horon tage\n"
const AdvicePassingHourBarMonth = "Ni parolu pli ol 28 horojn monate\n"
const AdvicePassingHourBarYear = "Ni parolu pli ol 365 horojn jare\n"
const AdvicePassingHourBar = AdvicePassingHourBarDay + AdvicePassingHourBarMonth + AdvicePassingHourBarYear
// basic advice info (CaptionHashTag to identify voice message)
const AdviceStart = "Konsiletoj:\n"
const CaptionHashtag = "#mezvenka"
const AdviceFileDescription = "Dosieroj estu kun " + CaptionHashtag + "\n"
const AdviceVoiceMsgBot = "@sonbonigisto_bot forigas silenton\n"
const AdviceCalendarReform = "Vizitu na kalendaro.xyz\n"
// full advice info to show
const Advice = AdviceStart + AdvicePassingHourBar + AdviceFileDescription + AdviceVoiceMsgBot + AdviceCalendarReform
// compact durations
const SecondWait = 1 * time.Second
const FiveSecondsWait = 5 * time.Second
const MinuteWait = 60 * time.Second

// stores duration data in config (day, month, year)
type Config struct {
	SpeakersDuration map[string]uint32 `json:"speakersDuration"`
	DurationTotal uint32             `json:"durationTotal"`
}

// stores duration datum for each speaker
type DurationDatum struct {
	SpeakerName string
	Duration    uint32
}

// stores date info
type DateInfo struct {
	Year int
	Month int
	MonthName string
	Day int
}

// loads initialization config
func loadInitConfig(filename string) (string, int64) {
	// set config to get
	type InitConfig struct {
		KeyAPI string
		ChatID int64
	}
	var initConfig InitConfig
	// get config or panic
	if data, err := os.ReadFile(filename); err != nil {
		panic(err)
	} else {
		json.Unmarshal(data, &initConfig)
	}
	return initConfig.KeyAPI, initConfig.ChatID
}

// updates duration data in memory
func updateDuration(msg *tgbotapi.Message, speakersDuration map[string]uint32, durationTotal *uint32) string {
	duration := 0
	// log if duration changes
	log := make([]string, UpdateLogNum)
	logMessage := ""
	// voice
	if msg.Voice != nil {
		duration = msg.Voice.Duration
	// video note
	} else if msg.VideoNote != nil {
		duration = msg.VideoNote.Duration
	// audio with CaptionHashtag
	} else if msg.Audio != nil {
		if msg.Caption != "" && strings.Contains(msg.Caption, CaptionHashtag) {
			duration = msg.Audio.Duration
		}
	// video with CaptionHashtag
	} else if msg.Video != nil {
		if msg.Caption != "" && strings.Contains(msg.Caption, CaptionHashtag) {
			duration = msg.Video.Duration
		}
	}
	// update duration data
	if duration > 0 {
		// log total duration data
		log[0] = fmt.Sprintf("%d+%d", *durationTotal, duration)
		*durationTotal += uint32(duration)
		log[1] = fmt.Sprintf("=%d", *durationTotal)
		// get the most precize name
		var speakerName string
		userName := msg.From.UserName
		firstName := msg.From.FirstName
		// @user
		if userName != "" {
			speakerName = "@" + userName
		// anonim
		} else if firstName != "" {
			speakerName = firstName
		}
		// update speaker duration data and log it
		if msg.From != nil && speakerName != "" {
			log[2] = fmt.Sprintf(" from %s[%d+%d=", speakerName, speakersDuration[speakerName], duration)
			speakersDuration[speakerName] += uint32(duration)
			log[3] = fmt.Sprintf("%d]", speakersDuration[speakerName])
		}
		for i := 0; i < UpdateLogNum; i++ {
			logMessage += log[i]
		}
	}
	return logMessage
}

// calculates duration time in hours, minutes, seconds
func calcDurationTime(durationTotal uint32) [3]int {
	duration := time.Duration(durationTotal) * time.Second
	hours := int(duration.Hours())
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	durationTime := [3]int{hours, minutes, seconds}
	return durationTime
}

// makes total duration presentable
func getDurationTotalInfo(durationTotal uint32) (string, int) {
	durationTotalTime := calcDurationTime(durationTotal)
	hours, minutes, seconds := durationTotalTime[0], durationTotalTime[1], durationTotalTime[2]
	// present info in human readable format
	durationTotalInfo := fmt.Sprintf("Voĉmesaĝdaŭro: %02dh %02dm %02ds\n\n", hours, minutes, seconds)
	return durationTotalInfo, hours
}

// makes speakers duration presentable
func getDurationSpeakersInfo(speakersDuration map[string]uint32) string {
	var durationSpeakersInfo string
	// only for maximum num of speakers
	speakersNum := SpeakersMaxNum
	if len(speakersDuration) < SpeakersMaxNum {
		speakersNum = len(speakersDuration)
	}
	// sort data based on duration
	durationData := make([]DurationDatum, speakersNum)
	for k, v := range speakersDuration {
		durationData = append(durationData, DurationDatum{SpeakerName: k, Duration: v})
	}
	sort.Slice(durationData, func(i, j int) bool {
		return durationData[i].Duration > durationData[j].Duration
	})
	// present info in human readable format
	for i := 0; i < speakersNum; i++ {
		speakerName := durationData[i].SpeakerName
		durationTime := calcDurationTime(durationData[i].Duration)
		hours, minutes, seconds := durationTime[0], durationTime[1], durationTime[2]
		durationSpeakersInfo += fmt.Sprintf("%d-as %s:\n%02dh %02dm %02ds\n", i+1, speakerName, hours, minutes, seconds)
	}
	durationSpeakersInfo += "\n"
	return durationSpeakersInfo
}

// makes duration presentable and show result
func getDurationInfo(timePeriod string, passingHourBarData map[string]int, speakersDuration map[string]uint32, durationTotal uint32) string {
	var result string
	var video string
	// set all video links
	videos := [4]string{BadVideo, GoodVideoOne, GoodVideoTwo, GoodVideoThree}
	// get all data for duration info
	durationTotalInfo, hours := getDurationTotalInfo(durationTotal)
	durationSpeakersInfo := getDurationSpeakersInfo(speakersDuration)
	passingHourBar := passingHourBarData[timePeriod]
	// bad result
	if hours < passingHourBar {
		result = BadResult
		video = videos[0]
	// good result
	} else if hours >= passingHourBar {
		result = GoodResult
		avgHours := hours / passingHourBar
		// specific video based on number of hours
		if avgHours >= 27 {
			video = videos[3]
		} else if avgHours >= 2 {
			video = videos[2] 
		} else if avgHours >= 1 {
			video = videos[1]
		}
	}
	// unify duration info
	durationInfo := durationTotalInfo + durationSpeakersInfo + result + video
	return durationInfo
}

// informs chat members about result
func summarize(bot *tgbotapi.BotAPI, ChatID int64, timePeriod string, configNameData map[string]string, passingHourBarData map[string]int, dateSummary string) {
	// get all data for summary
	configName := configNameData[timePeriod]
	speakersDuration, durationTotal := loadConfig(configName)
	participantsInfo := fmt.Sprintf("Voĉmesaĝistoj: %d\n", len(speakersDuration))
	durationInfo := getDurationInfo(timePeriod, passingHourBarData, speakersDuration, durationTotal)
	// unify summary
	summary := Hashtag + dateSummary + TimeExplanation + participantsInfo + durationInfo
	// send summary message
	msg := tgbotapi.NewMessage(ChatID, summary)
	for {
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
			time.Sleep(FiveSecondsWait)
		} else {
			break
		}
	}
}

// gives advice to chat members
func advise(bot *tgbotapi.BotAPI, ChatID int64) {
	// send advice message
	msg := tgbotapi.NewMessage(ChatID, Advice)
	for {
		_, err := bot.Send(msg)
		if err != nil {
			log.Println(err)
			time.Sleep(FiveSecondsWait)
		} else {
			break
		}
	}
}

// loads data from config
func loadConfig(filename string) (map[string]uint32, uint32) {
	var config Config
	var speakersDuration map[string]uint32
	var durationTotal uint32
	file, _ := os.OpenFile(filename, os.O_RDWR|os.O_CREATE, 0644)
	file.Close()
	data, _ := os.ReadFile(filename)
	json.Unmarshal(data, &config)
	// informs if config has been loaded or created
	defer log.Printf("%s config -> memory\n", filename)
	if config.SpeakersDuration != nil {
		speakersDuration = config.SpeakersDuration
	} else {
		speakersDuration = map[string]uint32{}
		log.Printf("0 -> %s config\n", filename)
	}
	durationTotal = config.DurationTotal
	return speakersDuration, durationTotal
}

// saves data to config
func saveConfig(filename string, speakersDuration map[string]uint32, durationTotal uint32) {
	config := Config{
		SpeakersDuration: speakersDuration,
		DurationTotal: durationTotal,
	}
	jsonData, _ := json.Marshal(config)
	os.WriteFile(filename, jsonData, 0644)
}

// updates config from another config
func updateConfig(timePeriod string, timePeriodUpdated string, configNameData map[string]string) {
	// get config names
	configName := configNameData[timePeriod]
	configUpdatedName := configNameData[timePeriodUpdated]
	// log update start and end
	log.Printf("%s config -> %s config […]\n", configName, configUpdatedName)
	defer log.Printf("%s config -> %s config [✓]\n", configName, configUpdatedName)
	// get new and old data
	speakersDuration, durationTotal := loadConfig(configName)
	speakersDurationOld, durationTotalOld := loadConfig(configUpdatedName)
	// update old data
	for k, v := range speakersDuration {
		speakersDurationOld[k] += v
	}
	durationTotalOld += durationTotal
	// save updated old data
	saveConfig(configUpdatedName, speakersDurationOld, durationTotalOld)
}

// resets memory and config data
func resetConfig(timePeriod string, configNameData map[string]string) {
	// get config name
	configName := configNameData[timePeriod]
	// log reset start and end
	log.Printf("%s config <- 0 […]\n", configName)
	defer log.Printf("%s config <- 0 [✓]\n", configName)
	// set empty variables
	speakersDuration := make(map[string]uint32)
	var durationTotal uint32 = 0
	// reset config
	saveConfig(configName, speakersDuration, durationTotal)
}

func main() {
	// initialize bot
	KeyAPI, ChatID := loadInitConfig("token.json")
	bot, err := tgbotapi.NewBotAPI(KeyAPI)
	if err != nil {
		log.Panic(err)
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 25
	updates := bot.GetUpdatesChan(u)
	// initalize all data variables
	configNameData := map[string]string{
		"day": DayJSON,
		"month": MonthJSON,
		"year": YearJSON,
	}
	passingHourBarData := map[string]int{
		"day": PassingHourBarDay,
		"month": PassingHourBarMonth,
		"year": PassingHourBarYear,
	}
	// initialize mutex
	var mutex sync.Mutex
	// initalize channel to reload config
	reload := make(chan bool, 1)
	// every 25 seconds check for new message
	go func(reload <-chan bool) {
		// load previous day data
		timePeriod := "day"
		configName := configNameData[timePeriod]
		mutex.Lock()
		speakersDuration, durationTotal := loadConfig(configName)
		mutex.Unlock()
		for update := range updates {
			// reload data day config on signal
			select {
			case <- reload:
				mutex.Lock()
				speakersDuration, durationTotal = loadConfig(configName)
				mutex.Unlock()
			// every message update duration and save day config
			default:
				msg := update.Message
				if msg != nil {
					logMessage := updateDuration(msg, speakersDuration, &durationTotal)
					if logMessage != "" {
						log.Println(logMessage)
					}
					mutex.Lock()
					saveConfig(configName, speakersDuration, durationTotal)
					mutex.Unlock()
				}
			}
		}
	}(reload)
	// every second check for time period passing
	dateInfo := ifc.GetDateInfo(0)
	previousDateInfo := dateInfo
	go func(reload chan<- bool) {
		for {
			newDate := false
			var timePeriod string
			var timePeriodUpdated string
			var dateSummary string
			// get current date info
			dateInfo = ifc.GetDateInfo(0)
			day, month, year := dateInfo.Day, dateInfo.Month, dateInfo.Year
			// get previous date info 
			previousDay, previousMonth, previousYear := previousDateInfo.Day, previousDateInfo.Month, previousDateInfo.Year
			previousMonthName := previousDateInfo.MonthName
			// every day update month config, summarize and reset day config
			if day != previousDay {
				dateSummary = fmt.Sprintf(PreviousDaySummary, previousDay, previousMonthName, previousYear)
				newDate = true
				timePeriod, timePeriodUpdated = "day", "month"
				summarize(bot, ChatID, timePeriod, configNameData, passingHourBarData, dateSummary)
				mutex.Lock()
				updateConfig(timePeriod, timePeriodUpdated, configNameData)
				resetConfig(timePeriod, configNameData)
				mutex.Unlock()
				log.Printf(NewDayLog, day, month, year)
				time.Sleep(SecondWait)
			}
			// every month update year config, summarize and reset month config
			if month != previousMonth {
				dateSummary = fmt.Sprintf(PreviousMonthSummary, previousMonthName, previousYear)
				newDate = true
				timePeriod, timePeriodUpdated = "month", "year"
				summarize(bot, ChatID, timePeriod, configNameData, passingHourBarData, dateSummary)
				mutex.Lock()
				updateConfig(timePeriod, timePeriodUpdated, configNameData)
				resetConfig(timePeriod, configNameData)
				mutex.Unlock()
				log.Printf(NewMonthLog, month, year)
				time.Sleep(SecondWait)
			}
			// every year summarize and reset year config
			if year != previousYear {
				dateSummary = fmt.Sprintf(PreviousYearSummary, previousYear)
				newDate = true
				timePeriod = "year"
				mutex.Lock()
				summarize(bot, ChatID, timePeriod, configNameData, passingHourBarData, dateSummary)
				resetConfig(timePeriod, configNameData)
				mutex.Unlock()
				log.Printf(NewYearLog, year)
				time.Sleep(SecondWait)
			}
			// every new date advise, reload data day config and sleep for a minute
			if newDate {
				advise(bot, ChatID)
				reload <- true
				time.Sleep(MinuteWait)
			}
			previousDateInfo = dateInfo
			time.Sleep(SecondWait)
		}
	}(reload)
	select {}
}
