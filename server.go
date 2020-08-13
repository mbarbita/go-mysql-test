package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"text/template"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/websocket"

	cfgutils "github.com/mbarbita/golib-cfgutils"
)

// handle func / ; /index.html ; /home
func home(w http.ResponseWriter, r *http.Request) {

	// parse templates
	htmlTpl := template.Must(template.ParseGlob("templates/*.*"))
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
	htmlTpl := template.Must(template.ParseGlob("templates/*.*"))
	fmt.Println("Templates:", htmlTpl.DefinedTemplates())
	fmt.Println("Tpl Name:", htmlTpl.Name())

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

	rows, err := db.Query("select id, name from users where id = ?", 2)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var data struct {
		ID   int
		Name string
	}

	for rows.Next() {
		// err := rows.Scan(&id, &name)
		err := rows.Scan(&data.ID, &data.Name)
		if err != nil {
			panic(err)
		}
		// log.Println(id, name)
		log.Println(data)
	}
	err = rows.Err()
	if err != nil {
		panic(err)
	}

	// var buffer bytes.Buffer
	// var namest []string
	// namest = append(namest, "1", "unu")

	// // json.NewEncoder(&buffer).Encode(&name)
	// json.NewEncoder(&buffer).Encode(&namest)

	// fmt.Println("\nUsing Encoder:\n" + buffer.String())

	// data := struct {
	// 	ID   int
	// 	Name string
	// }{
	// 	1,
	// 	"unu",
	// }
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

	// type ColorGroup struct {
	// 	ID     int
	// 	Name   string
	// 	Colors []string
	// }
	// group := ColorGroup{
	// 	ID:     1,
	// 	Name:   "Reds",
	// 	Colors: []string{"Crimson", "Red", "Ruby", "Maroon"},
	// }

	// b, err := json.Marshal(group)
	// if err != nil {
	// 	fmt.Println("error:", err)
	// }
	// // os.Stdout.Write(b)
	// fmt.Println(string(b))

	// err = c.WriteMessage(1, b) // message type = 1
	// if err != nil {
	// 	log.Println("ws write err:", err)
	// 	return
	// }

	// }

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
