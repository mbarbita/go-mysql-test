package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"

	"github.com/gorilla/websocket"

	cfgutils "github.com/mbarbita/golib-cfgutils"
)

// handle func /
func home(w http.ResponseWriter, r *http.Request) {

	// parse templates
	htmlTpl := template.Must(template.ParseGlob("templates/*.*"))
	fmt.Println("Templates:", htmlTpl.DefinedTemplates())
	fmt.Println("Tpl Name:", htmlTpl.Name())

	// data for template
	tplData := r.Host
	fmt.Println(tplData)

	// Execute template
	err := htmlTpl.ExecuteTemplate(w, "index.html", tplData)
	// err := htmlTpl.Execute(w, tplData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

// handle func /msg
func wsMessage(w http.ResponseWriter, r *http.Request) {

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
	log.Println("wsm after:", wsm)

	// response := []byte("response")
	resp := WsOut{"kill", "bill"}
	response, err := json.Marshal(resp)
	if err != nil {
		log.Println("marshal err:", err)
	}
	log.Println("response after:", string(response))

	err = c.WriteMessage(1, response) // message type = 1
	if err != nil {
		log.Println("ws write err:", err)
		return
	}

	log.Println("ws sent response")
}

var cfgMap map[string]string

func main() {

	cfgMap = cfgutils.ReadCfgFile("cfg.ini", false)

	http.HandleFunc("/", home)
	http.HandleFunc("/msg", wsMessage)

	log.Println("Server listening on:", cfgMap["server"])
	err := http.ListenAndServe(cfgMap["server"], nil)
	if err != nil {
		panic("ListenAndServe err: " + err.Error())
	}
}
