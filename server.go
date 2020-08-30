package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"github.com/fsnotify/fsnotify"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"

	cfgutils "github.com/mbarbita/golib-cfgutils"
	// _ "net/http/pprof"
)

// handle func / ; /index.html ; /home
func home(w http.ResponseWriter, r *http.Request) {

	// The "/" pattern matches everything, so we need to check
	// that we're at the root here.
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// parse templates
	// htmlTpl := template.Must(template.ParseGlob("templates/*.*"))
	// fmt.Println("Templates:", htmlTpl.DefinedTemplates())
	// fmt.Println("Tpl Name:", htmlTpl.Name())

	// data for template
	var tplData = make(map[string]string)
	tplData["pagename"] = "home"
	// fmt.Fprintf(w, "r.Host %v\n", r.Host)
	// fmt.Fprintf(w, "r.URL.Path %v\n", r.URL.Path)

	// Execute template
	err := htmlTpl.ExecuteTemplate(w, "home-page.html", tplData)
	// err := htmlTpl.Execute(w, tplData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

// handle func /test
func test(w http.ResponseWriter, r *http.Request) {

	// parse templates
	// htmlTpl := template.Must(template.ParseGlob("templates/*.*"))
	// fmt.Println("Templates:", htmlTpl.DefinedTemplates())
	// fmt.Println("Tpl Name:", htmlTpl.Name())

	// data for template
	var tplData = make(map[string]string)
	tplData["pagename"] = "test"
	tplData["wshost1"] = "ws://" + r.Host + "/testmsg"
	tplData["wsshost1"] = "wss://" + r.Host + "/testmsg"
	fmt.Println(tplData)

	// Execute template
	err := htmlTpl.ExecuteTemplate(w, "test-page.html", tplData)
	// err := htmlTpl.Execute(w, tplData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

// handle func /testmsg
func testmsg(w http.ResponseWriter, r *http.Request) {

	type WsIn struct {
		Fa string
		Fb string
		Fc float64
	}

	type WsOut struct {
		Fd string
		Fe string
	}

	var upgrader = websocket.Upgrader{} // use default options
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade:", err)
		return
	}
	defer c.Close()

	_, message, err := c.ReadMessage()
	if err != nil {
		log.Println("read:", err)
		return
	}

	log.Printf("recv: %s", message)
	var wsm WsIn

	// Only exported fields of target struct gets decoded (Capital First Letter)
	// log.Println("wsm before:", wsm)
	err = json.Unmarshal(message, &wsm)
	if err != nil {
		log.Println("unmarshal err:", err)
	}
	log.Println("wsm (unmarshal):", wsm)

	// response := []byte("response")
	resp := WsOut{"kill", "bill"}
	response, err := json.Marshal(resp)
	if err != nil {
		log.Println("marshal err:", err)
	}
	log.Println("response (marshal):", string(response))

	err = c.WriteMessage(1, response) // message type = 1
	if err != nil {
		log.Println("ws write err:", err)
		return
	}

	log.Println("ws sent response")

	//test db
	// func dbtest() {

	// var (
	// 	id   int
	// 	name string
	// )

	db, err := sql.Open("mysql", cfgMap["DSN"])
	if err != nil {
		panic(err) // Just for example purpose. You should use proper error handling instead of panic
	}
	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	log.Println("Pinging...")
	err = db.Ping()
	if err != nil {
		panic(err) // proper error handling instead of panic in your app
	}
	log.Println("Pinging OK")

	// rows, err := db.Query("select id, name from users where id = ?", 2)
	rows, err := db.Query("select * from users")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var data struct {
		ID   int
		Name string
	}

	for rows.Next() {
		err := rows.Scan(&data.ID, &data.Name)
		if err != nil {
			panic(err)
		}
		// log.Println(id, name)
		log.Println(data)

		json, err := json.Marshal(data)
		if err != nil {
			panic(err)
		}

		fmt.Println(string(json))

		err = c.WriteMessage(1, json) // message type = 1
		if err != nil {
			log.Println("ws write err:", err)
			return
		}
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}
}

func dirWatcher(folders ...string) {
	time.Sleep(5 * time.Second)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		var lock sync.Mutex

		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				log.Println("event:", event)
				if event.Op&fsnotify.Write == fsnotify.Write {
					log.Println("modified file:", event.Name)
					// parse templates
					lock.Lock()
					htmlTpl = template.Must(template.ParseGlob("templates/*.*"))
					fmt.Println("Templates:", htmlTpl.DefinedTemplates())
					fmt.Println("Tpl Name:", htmlTpl.Name())
					lock.Unlock()
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Println("error:", err)
			}
		}
	}()
	for _, folder := range folders {
		err = watcher.Add(folder)
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Added folder to watchlist:", folder)
	}
	wg.Wait()
}

var cfgMap map[string]string
var htmlTpl *template.Template

func main() {

	// set dual log
	dualLog := flag.Bool("logtofile", false, "log to flie too")
	flag.Parse()

	if *dualLog {
		// If the dir doesn't exist, create it
		if _, err := os.Stat("log"); os.IsNotExist(err) {
			log.Println("log dir does not exist, creating...")
			err := os.MkdirAll(filepath.Join("log"), os.ModeDir)
			if err != nil {
				log.Fatal(err)
			}
		}

		// If the file doesn't exist, create it, or append to the file
		logFileName := "log/" + time.Now().Format("2006-01-02 150405") + ".log"
		logFile, err := os.OpenFile(logFileName,
			os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer logFile.Close()
		wrt := io.MultiWriter(os.Stdout, logFile)
		log.SetOutput(wrt)
		log.Println("Dual log output ON")
		log.Println("log file:", logFileName)

	}

	cfgMap = cfgutils.ReadCfgFile("cfg.ini", false)

	// parse templates
	htmlTpl = template.Must(template.ParseGlob("templates/*.*"))
	fmt.Println("Templates:", htmlTpl.DefinedTemplates())
	fmt.Println("Tpl Name:", htmlTpl.Name())

	go dirWatcher("templates")

	//routes
	http.HandleFunc("/", home)
	http.HandleFunc("/test", test)
	http.HandleFunc("/testmsg", testmsg)

	http.Handle("/download/", http.StripPrefix("/download/",
		http.FileServer(http.Dir("download"))))

	go func() {
		log.Println("TLS Server listening on:", cfgMap["serverTLS"])
		err := http.ListenAndServeTLS(cfgMap["serverTLS"], "pki/server.crt", "pki/server.key", nil)
		if err != nil {
			panic("ListenAndServeTLS: " + err.Error())
		}
	}()

	log.Println("Server listening on:", cfgMap["server"])
	err := http.ListenAndServe(cfgMap["server"], nil)
	if err != nil {
		panic("ListenAndServe err: " + err.Error())
	}
}
