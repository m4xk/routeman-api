package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/jackpal/gateway"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

const CONFIG_FILE string = "./config.json"

type Gateway struct {
	Name      string
	IpAddress string
}

type Config struct {
	Listen     string
	AllowedIps []string
	Gateways   []Gateway
}

type Response struct {
	Status string `json:"status,omitempty"`
	Data   string `json:"data,omitempty"`
}

var config Config

func readConfig() {

	jsonFile, err := os.Open(CONFIG_FILE)

	if err != nil {
		fmt.Println(err)
	}

	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	jsonErr := json.Unmarshal([]byte(byteValue), &config)

	if jsonErr != nil {
		panic(jsonErr)
	}
}

func main() {
	readConfig()

	router := mux.NewRouter()

	router.Use(authMiddleware)
	router.Use(commonMiddleware)

	router.HandleFunc("/gateways/current", getCurrentDefaultRoute).Methods("GET")
	router.HandleFunc("/gateways/{gw}/activate", activateGw).Methods("GET")

	log.Fatal(http.ListenAndServe(config.Listen, router))
}

func authMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		slice := strings.Split(r.RemoteAddr, ":")

		for _, v := range config.AllowedIps {
			if v == slice[0] {
				next.ServeHTTP(w, r)

				return
			}
		}

		http.Error(w, "Forbidden", http.StatusForbidden)
	})
}

func commonMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		next.ServeHTTP(w, r)
	})
}

func flushDefaultRoutes() bool {

	cmd := exec.Command("ip", "route", "flush", "0/0")

	var out bytes.Buffer

	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		return true
	} else {
		return false
	}
}

func getCurrentDefaultRoute(w http.ResponseWriter, r *http.Request) {
	ip, _ := gateway.DiscoverGateway()

	json.NewEncoder(w).Encode(Response{Status: "success", Data: gwNameToIp(ip.String())})
}

func gwIpToName(gwname string) string {

	for _, v := range config.Gateways {
		if gwname == v.Name {
			return v.IpAddress
		}
	}

	return ""
}

func gwNameToIp(ip string) string {

	for _, v := range config.Gateways {
		if ip == v.IpAddress {
			return v.Name
		}
	}

	return ""
}

func activateGw(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)

	route := gwIpToName(vars["gw"])

	if len(route) == 0 {
		json.NewEncoder(w).Encode(Response{Status: "fail", Data: "Non-existent route"})

		return
	}

	flushDefaultRoutes()

	cmd := exec.Command("ip", "route", "add", "default", "via", route)

	var out bytes.Buffer

	cmd.Stdout = &out
	err := cmd.Run()

	if err != nil {
		log.Fatal(err)
	}

	json.NewEncoder(w).Encode(Response{Status: "success", Data: ""})
}
