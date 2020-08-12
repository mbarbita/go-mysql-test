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

// handle func / ; /index.html ; /home
func home(w http.ResponseWriter, r *http.Request) {
	// fmt.Fprintf(w, "r.Host %v\n", r.Host)
	// fmt.Fprintf(w, "r.URL.Path %v\n", r.URL.Path)
	htmlTpl := template.Must(template.ParseGlob("templates/*.*"))
	fmt.Println("Templates:", htmlTpl.DefinedTemplates())
	fmt.Println("Tpl Name:", htmlTpl.Name())

	// Execute template
	err := htmlTpl.ExecuteTemplate(w, "generic-page.html", nil)
	// err := htmlTpl.Execute(w, tplData)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError),
			http.StatusInternalServerError)
	}
}

// handle func /test
func test(w http.ResponseWriter, r *http.Request) {

	// parse templates
	htmlTpl := template.Must(template.ParseGlob("templates/*.*"))
	fmt.Println("Templates:", htmlTpl.DefinedTemplates())
	fmt.Println("Tpl Name:", htmlTpl.Name())

	// data for template
	var tplData = []string{"ws://" + r.Host + "/testmsg"}
	fmt.Println(tplData)

	// Execute template
	err := htmlTpl.ExecuteTemplate(w, "test-page.html", tplData[0])
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
}

var cfgMap map[string]string

func main() {

	cfgMap = cfgutils.ReadCfgFile("cfg.ini", false)

	//routes
	http.HandleFunc("/", home)
	http.HandleFunc("/index.html", home)
	http.HandleFunc("/home", home)
	http.HandleFunc("/test", test)
	http.HandleFunc("/testmsg", testmsg)

	log.Println("Server listening on:", cfgMap["server"])
	err := http.ListenAndServe(cfgMap["server"], nil)
	if err != nil {
		panic("ListenAndServe err: " + err.Error())
	}
}
