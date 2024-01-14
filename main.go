package main

import (
    "context"
    "sync"
    //"errors"
    "reflect"
    "bufio"
    "time"
    "fmt"
	"os"
    "regexp"
    "strings"
    "math/rand"
    "encoding/json"

    "io/fs"
    "log"

    errs2 "gopkg.in/src-d/go-errors.v1"


    "github.com/pbnjay/memory"
	_ "github.com/mattn/go-sqlite3"

    "maunium.net/go/mautrix"
    event "maunium.net/go/mautrix/event"
    "maunium.net/go/mautrix/crypto/cryptohelper"
    id "maunium.net/go/mautrix/id"

    "github.com/gocolly/colly/v2"
)

var StackPrintError = errs2.NewKind("Error! vVv Stack trace vVv")

//possibly could further abstract these
func LogFCheck(e error){
    if e != nil{
        msg := fmt.Sprintf("%+v\n%+v", StackPrintError.New(), e)
        log.Fatal(msg)
    }
}

func PrintCheck(e error){
    if e != nil{
        msg := fmt.Sprintf("%+v\n%+v", StackPrintError.New(), e)
        log.Println(msg)
    }
}

func PanicCheck(e error){
    if e != nil{
        msg := fmt.Sprintf("%+v\n%+v", StackPrintError.New(), e)
        log.Panicln(msg)
    }
}

type FreeMemError struct {
    FreeMem float64
    FileSize float64
}

func (e *FreeMemError) Error() string {
    return fmt.Sprintf("Error: Free memory is %+v , but file size is %+v. This has been arbitrarily chosen as an unacceptable portion of available free memory for this operation.", e.FreeMem, e.FileSize)
}

func CheckMemConditionsPreFileLoad( acceptableMemRatio float64 , fileInfoIn fs.FileInfo ) ( memConditionsGood bool , errOut error ){
        memConditionsGood = false
        freeMem := float64(memory.FreeMemory())
        freeMemTimesRatio := freeMem*acceptableMemRatio
        fileSizeAsFloat := float64(fileInfoIn.Size())
        fmt.Printf("Free system memory * accetableMemRatio: %+v\n", freeMemTimesRatio )
        fmt.Printf( "Size of %+v: %+v\n", fileInfoIn.Name(), fileSizeAsFloat )

        if fileSizeAsFloat < freeMemTimesRatio {
            fmt.Println("File size is within acceptable bounds.")
            memConditionsGood = true
            return
        } else {
            errOut = &FreeMemError{ FreeMem: freeMem , FileSize: fileSizeAsFloat}
            return
        }
}
//for unmarshalling the config file
type CredzNPathz struct {
    MatrixHomeserver string `json:"homeserver"`
    MatrixUser id.UserID `json:"matrixUser"`
    MatrixRoom id.RoomID `json:"matrixRoomID"`
    MatrixPassword string `json:"matrixPassword"`

    SQLiteDatabase string `json:"database"`
}

type ChannelMessage struct {
    Type int
    Text string
}

type PostData struct {
    VideoPaths []VideoInfo
    Word string
    WordURL string
    HowToSignString string
    Other []string
    Similar string
}

type PostQueue []PostData

type VideoInfo struct {
    Word string
    VideoURL string
    ThumbnailURL string
}

type VidlessPair struct {
    Word string
    URL string
}

func ( thisQueue PostQueue ) PrintQueue() {
    for _, entry := range( thisQueue ){
        fmt.Println(entry.Word)
        if entry.HowToSignString != "" {
            fmt.Println(entry.HowToSignString)
        }
        if len(entry.VideoPaths) != 0 {
            for _, vid := range(entry.VideoPaths){
                fmt.Println(vid)
            }
        }
        if len(entry.Other) != 0 {
            for _, thing := range(entry.Other){
                fmt.Println(thing)
            }
        }
        if entry.Similar != "" {
            fmt.Println(entry.Similar)
        }
    }
}

func ( thisQueue PostQueue ) MessageQueue( messagePrefix string ) {
    postBuilder := ""
    thisPrefix := fmt.Sprintf("<h1><strong><em>%s</em></strong></h1>", messagePrefix )
    //sendFormattedText(thisPrefix)
    postBuilder = fmt.Sprintf("%s\n",thisPrefix )
    for queueIndex , entry := range( thisQueue ){
        if queueIndex == 0 {
            thisWord := fmt.Sprintf("<p><strong><a href=%s>%s</a></strong> - signasl.org main page link</p>", entry.WordURL, entry.Word)
            //sendFormattedText(thisWord)
            postBuilder = fmt.Sprintf("%s\n%s\n",postBuilder, thisWord)
        }
        thisWord := fmt.Sprintf("<p><u>%s (Definition %v)</u></p>", entry.Word, queueIndex )
        //sendFormattedText(thisWord)
        postBuilder = fmt.Sprintf("%s\n%s\n", postBuilder, thisWord)
        if entry.HowToSignString != "" {
            thisHowToSignString := fmt.Sprintf("<p>%s</p>", entry.HowToSignString )
            postBuilder = fmt.Sprintf("%s\n%s\n", postBuilder, thisHowToSignString)
        }

        if len(entry.VideoPaths) != 0 {
            videoLinkListString := ""
            for index, vid := range(entry.VideoPaths){
                videoLinkListString = fmt.Sprintf( "%s<p><a href=%s>%s: Def %v Video %v</a></p>\n" , videoLinkListString ,vid.VideoURL, entry.Word, queueIndex, index )
            }
            postBuilder = fmt.Sprintf("%s\n%s\n", postBuilder, videoLinkListString )
        }
        if len(entry.Other) != 0 {
            for _, thing := range(entry.Other){
                postBuilder = fmt.Sprintf("%s\n%s\n", postBuilder, thing )
            }
        }
        if entry.Similar != "" {
            thisSimilarString := fmt.Sprintf("<p><em>%s</em></p>", entry.Similar )
            postBuilder = fmt.Sprintf("%s\n%s\n", postBuilder, thisSimilarString )
        }
    }
    suffix := clearVidlessWords()
    postBuilder = fmt.Sprintf("%s\n<p>%s</p>\n", postBuilder, suffix )
    sendFormattedText(postBuilder)
}

// globalz lol
var (
    configFile string
    urlsFilePath string

	cli *mautrix.Client
    matrixRoom id.RoomID
    matrixUser id.UserID

    currentWord string
    currentURL string
    definitionIndex int

    noVidsFound bool
    workingGlobalPost PostData
    globalPostQueue PostQueue

    globalFileLines []string
    globalVidlessWords []VidlessPair

    globalSleep time.Duration
)

func searchFileLines( searchTermIn []byte ) (matchesOut []string) {
    var err error
    statLoad( urlsFilePath )
    matches := []string{}
    urlMatchPrefix := "https://www.signasl.org/sign/"
    searchTermMatchPattern := fmt.Sprintf( `%s%v`, urlMatchPrefix, string(searchTermIn) )
    fmt.Println("searchTermMatchPattern:",searchTermMatchPattern)
    for _ , line := range( globalFileLines ) {
        //byteLine := []byte(line)
        //searchMatch , err := regexp.Match( searchTermMatchPattern, byteLine )
        searchMatch := line == searchTermMatchPattern
        if err != nil {
            fmt.Println("Error on search against: ", searchTermMatchPattern)
        }
        PrintCheck(err)
        if err == nil {
            if searchMatch{
                matches = append(matches, line)
            }
        }
        matchesOut = matches
    }
    return
}

func clearGlobalQueue( msgPrefix string ) {
    globalPostQueue.MessageQueue( msgPrefix )
    globalPostQueue = PostQueue{}
    definitionIndex = 0
}

func clearVidlessWords() string {
    workingEntry := "<u>Words tried before finding one with at least one video:</u>"
    if len(globalVidlessWords) != 0{
        for index, entry := range globalVidlessWords {
            if index == 0 {
                workingEntry = fmt.Sprintf("%s <a href=%s>%s</a>", workingEntry, entry.URL, entry.Word)
            } else {
                workingEntry = fmt.Sprintf("%s, <a href=%s>%s</a>", workingEntry, entry.URL, entry.Word)
            }
        }
        workingEntry = fmt.Sprintf("%s . Know one of these words? Consider uploading a video!", workingEntry)
    }
    if len(globalVidlessWords) == 0{
        workingEntry = fmt.Sprintf("%s %s", workingEntry, "None!")
    }
    globalVidlessWords = []VidlessPair{}
    return workingEntry
}


func parseEnv() {
	configFile = os.Getenv("MATRIX_DAILY_ASL_CONFIG_FILE")
    if configFile == "" {
        configFile = "manifest.json"
	}
}

func parseManifestJson ( jsonFilepath string ) ( manifestOut CredzNPathz ) {
    content, err := os.ReadFile( jsonFilepath )
    PanicCheck(err)
    err = json.Unmarshal( content, &manifestOut )
    PanicCheck(err)
    return
}

func sendFormattedText( text string ) {
    testStruct := event.MessageEventContent{
		MsgType: "m.text",
		Body:   text,
        Format: "org.matrix.custom.html",
        FormattedBody: text,
	}
	_, err := cli.SendMessageEvent(matrixRoom, event.EventMessage, &testStruct )
    PrintCheck(err)
    time.Sleep(globalSleep)
}

func SelectorTargetedCallback(i int, e *colly.HTMLElement) {
    if i == 0 {
        workingGlobalPost = PostData{}
    }
    thisText := e.Text
    if e.Name == "h1" || e.Name == "h2" {
        if !reflect.DeepEqual( workingGlobalPost , PostData{} ){
            definitionIndex = definitionIndex + 1
            addPostDataToQueue()
        }
        workingGlobalPost.Word = thisText
        workingGlobalPost.WordURL = currentURL
    }
    if e.Name == "video" {
        VideoElementCallback(0,e)
    }

    if thisText != "Your browser does not support HTML5 video."{
        byteText := []byte(thisText)
        howToSignMatch, err := regexp.Match( `How to sign:.*` , byteText )
        PrintCheck(err)
        if howToSignMatch {
            workingGlobalPost.HowToSignString = thisText
        }
        similarSameMatch, err := regexp.Match(`Similiar / Same:.*` , byteText )
        if similarSameMatch {
            workingGlobalPost.Similar = thisText
        }
    }
}

func PrintText(i int, e *colly.HTMLElement) {
    fmt.Println(e.Attr("src"))
}

func VisitSrc(i int, e *colly.HTMLElement) {
    thisSrc := e.Attr("src")
    e.Request.Visit(thisSrc)
}

func VideoElementCallback(i int, e *colly.HTMLElement) {
    //currentThumbnailURL := e.Attr("poster")
    e.ForEach("source[src]", VisitSrc )
}

func CheckForNoVid(i int, e *colly.HTMLElement) {
    thisText := e.Text
    noVidString := "Sorry, no video found for this word"
    thisVidlessPair := VidlessPair{ Word: currentWord , URL: currentURL }
    if strings.Contains(thisText, noVidString) {
        globalVidlessWords = append( globalVidlessWords , thisVidlessPair )
        noVidsFound = true
    }
}

func descendIntoColMd12( i int , e *colly.HTMLElement ){
    if e.Attr("class") == "col-md-12" {
        e.ForEach( "b" , CheckForNoVid )
        thisSelector := "h1,h2,p,video"
        e.ForEach(thisSelector, SelectorTargetedCallback)
    }
}

func addPostDataToQueue(){
    if !noVidsFound {
        globalPostQueue = append( globalPostQueue , workingGlobalPost )
    }
    workingGlobalPost = PostData{}
}

func ParsePageStructure(e *colly.HTMLElement) {
    currentWord = e.ChildText("h1")
    e.ForEach("div", descendIntoColMd12)
    addPostDataToQueue()
}

func LoadURLFileIntoMemory( file *os.File ) {
    fileLines := []string{}
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        line := scanner.Text()
        fileLines = append(fileLines, line)
    }
    globalFileLines = fileLines
}

func pickFromFileLines() ( urlOut string ) {
    randsource := rand.NewSource(time.Now().UnixNano())
    randgenerator := rand.New(randsource)
    urlOut = globalFileLines[ randgenerator.Intn( len(globalFileLines) ) ]
    return
}

func statLoad( filepathIn string ) ( errOut error ) {
    if globalFileLines != nil {
        errOut = nil
        return
    }
    fileInfo , err := os.Stat( filepathIn )
    PrintCheck(err)
    memConditionsGood, err := CheckMemConditionsPreFileLoad( 1.0/40.0 , fileInfo )
    if err == nil && memConditionsGood {
        file, err := os.Open(filepathIn)
        defer file.Close()
        if err == nil {
            LoadURLFileIntoMemory( file )
            errOut = nil
            return
        }
    }
    return
}

//Check to see that the source file (contains signasl word URLs) is small enough
// to not overwhelm the current available memory (is < freememory/40). If so,
// load the whole file into memory to pick a word.
//TODO: Implement a more efficient scheme to pick a random line.
func StatOpenPick( filepathIn string ) ( urlOut string , errOut error ) {
    errOut = statLoad( filepathIn )
    if errOut == nil {
        urlOut = pickFromFileLines()
    }
    return
}

func VidResponseCallback(r *colly.Response) {
    responseURL := r.Request.URL.String()
    thisVidInfo := VideoInfo { Word: currentWord, VideoURL: responseURL}
    if r.Headers.Get("Content-Type") == "video/mp4" {
        noVidsFound = false
        workingGlobalPost.VideoPaths = append( workingGlobalPost.VideoPaths , thisVidInfo )
    }
}

func SendDailyPostSignal( sleepTimeIn time.Duration , signalChan chan int , conditionIn bool ) {
    for conditionIn {
        signalChan <- 1
        time.Sleep( sleepTimeIn )
    }
}

func main() {

	parseEnv()
	var err error
    urlsFilePath = "signasl_all_unique.txt"
    globalSleep = 2500 * time.Millisecond

    thisCondition := true
    dailySleepTime := 24*60*60 * time.Second
    dailySignalChan := make(chan int , 2)
    //var event string
    manifest := parseManifestJson( configFile )
    fmt.Println("homeserver: ", manifest.MatrixHomeserver)
    fmt.Println("user: ", manifest.MatrixUser)
    matrixRoom = manifest.MatrixRoom
    matrixUser = manifest.MatrixUser
	// Instantiate default collector
//Colly setup
	c := colly.NewCollector(
		// Visit only domains: signasl.org, www.signasl.org
		colly.AllowedDomains("signasl.org", "www.signasl.org","media.signbsl.com"),

		// Cache responses to prevent multiple download of pages
		// even if the collector is restarted
		colly.CacheDir("./signASL_cache"),
	)
    c.OnHTML("body", ParsePageStructure )
    c.OnResponse( VidResponseCallback )



//Mautrix setup
	cli, err = mautrix.NewClient(manifest.MatrixHomeserver, "","")
    PanicCheck(err)

    queueChannel := make(chan ChannelMessage, 1000)

    syncer := cli.Syncer.(*mautrix.DefaultSyncer)
	syncer.OnEventType(event.EventMessage, func(source mautrix.EventSource, evt *event.Event) {
        thisMessageBody := fmt.Sprintf( "%v" , evt.Content.AsMessage().Body)
        byteMessageBody := []byte(thisMessageBody)
        anotherMatch, err := regexp.Match( `!asl another.*` , byteMessageBody )
        PrintCheck(err)
        if err == nil && anotherMatch {
            thisMessage := ChannelMessage{ Type: 1 , Text: "" }
            queueChannel <- thisMessage
            time.Sleep(globalSleep)
        }
        searchMatch, err := regexp.Match( `!asl search .*` , byteMessageBody )
        PrintCheck(err)
        if searchMatch {
            searchTerm := strings.Split( thisMessageBody, "!asl search ")[1]
            searchMsg := fmt.Sprintf("Searching for \"%v\" ...\n", searchTerm )
            sendFormattedText(searchMsg)
            matchesFound := searchFileLines( []byte(searchTerm) )
            numMatches := len(matchesFound)
            if numMatches == 0 {
                sendFormattedText("No Matches found!")
            }
            if numMatches >= 4 {
                postBuilder := fmt.Sprintf("%v matches found, too many to summon.", numMatches)
                limitInd := 0
                for limitInd < 20 {
                    postBuilder = fmt.Sprintf("%v %v", postBuilder, matchesFound[limitInd])
                    limitInd = limitInd + 1
                }

                sendFormattedText(postBuilder)
            }
            if numMatches  < 4 {
                for _, match := range(matchesFound) {
                    thisMessage := ChannelMessage{ Type: 2  , Text: match }
                    queueChannel <- thisMessage
                    time.Sleep(globalSleep)
                }
            }
        }
        helpMatch, err := regexp.Match( `!asl help.*` , byteMessageBody )
        PrintCheck(err)
        if err == nil && helpMatch {
                helpMsg := fmt.Sprintf("Commands:\n!asl another --get a random word\n !asl search <INSERT WORD> --search")
                sendFormattedText(helpMsg)
                time.Sleep(globalSleep)
        }
	})
    
    userLocalPart := string(manifest.MatrixUser)
	cryptoHelper, err := cryptohelper.NewCryptoHelper(cli, []byte("meow"), manifest.SQLiteDatabase)
	cryptoHelper.LoginAs = &mautrix.ReqLogin{
		Type:       mautrix.AuthTypePassword,
		Identifier: mautrix.UserIdentifier{Type: mautrix.IdentifierTypeUser, User: userLocalPart},
		Password:   manifest.MatrixPassword,
	}
    PanicCheck(err)
	err = cryptoHelper.Init()
    PanicCheck(err)
    cli.Crypto = cryptoHelper

	syncCtx, _ := context.WithCancel(context.Background())
	var syncStopWait sync.WaitGroup
	syncStopWait.Add(1)

	go func() {
		err = cli.SyncWithContext(syncCtx)
		defer syncStopWait.Done()
		if err != nil && !errors.Is(err, context.Canceled) {
			panic(err)
		}
	}()



	syncStopWait.Add(1)
    go SendDailyPostSignal( dailySleepTime , dailySignalChan , thisCondition )

//Main bot loop starts here
    someInd := 0
    definitionIndex = 0
    for {
        select {
            case chatSignal := <- queueChannel:
                if chatSignal.Type == 1 {
                    noVidsFound = true
                    for noVidsFound {
                            thisURL, err := StatOpenPick( urlsFilePath )
                            if err == nil {
                                currentURL = thisURL
                                c.Visit(thisURL)
                            }
                    }
                    if len(globalPostQueue) > 0 {
                        clearGlobalQueue( "Another random word, as requested!" )
                    }
                    someInd = someInd + 1
                    time.Sleep(globalSleep)
                }
                if chatSignal.Type == 2 {
                    thisURL := chatSignal.Text
                    currentURL = thisURL
                    c.Visit(thisURL)
                    if len(globalPostQueue) > 0 {
                        thisMsg := "The only match: "
                        clearGlobalQueue( thisMsg )
                    }
                    time.Sleep(globalSleep)

                }

            case chatSignal := <- dailySignalChan:
                if chatSignal == 1 {
                    noVidsFound = true
                    for noVidsFound {
                            thisURL, err := StatOpenPick( urlsFilePath )
                            if err == nil {
                                currentURL = thisURL
                                c.Visit(thisURL)
                            }
                    }
                    if len(globalPostQueue) > 0 {
                        clearGlobalQueue( "Word of the Day!" )
                    }
                    someInd = someInd + 1
                    time.Sleep(globalSleep)
                }
        }
        time.Sleep(globalSleep)
    }

    err = cryptoHelper.Close()
    LogFCheck(err)
}

