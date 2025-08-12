package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

type Alert_Request struct {
	Status string `json:"status"`
	Alerts []struct {
		Status string            `json:"status"`
		Labels map[string]string `json:"labels"`
	} `json:"alerts"`
}

func alert_webhook(w http.ResponseWriter, r *http.Request) {
	var req Alert_Request
	j, _ := io.ReadAll(r.Body)
	json.Unmarshal(j, &req)
	fmt.Printf("%v\n", req)

	mode := [7]int{0, 0, 0, 0, 1, 0, 0}
	duration := 10 * time.Second
	buzzSpecified := false

	if len(req.Alerts) > 0 {
		alert := req.Alerts[0]
		severity := alert.Labels["severity"]
		beepMode := alert.Labels["beep"]

		// Set blinking and buzzer mode
		switch severity {
		case "critical":
			mode = [7]int{0, 1, 0, 0, 0, 0, 0} // Red blink
		case "warning":
			mode = [7]int{1, 0, 0, 0, 0, 0, 0} // Yellow blink
		}

		// Buzzer
		switch beepMode {
		case "long":
			mode[2] = 1 // Beep 1s
			buzzSpecified = true
		case "short":
			mode[3] = 1 // Beep short
			buzzSpecified = true
		}

		if buzzStr, ok := alert.Labels["buzztime"]; ok {
			var buzzSec int
			_, err := fmt.Sscanf(buzzStr, "%d", &buzzSec)
			if err == nil && buzzSec > 0 {
				duration = time.Duration(buzzSec) * time.Second
			}
		} else if buzzSpecified {
			duration = 10 * time.Second
		}
	}

	SetPatliteMode(mode)

	go func() {
		time.Sleep(duration)
		SetPatliteMode([7]int{0, 0, 0, 0, 1, 0, 0})
	}()
}

func main() {
	SetPatliteMode([7]int{0, 0, 0, 0, 1, 0, 0})

	http.HandleFunc("/alert_webhook", alert_webhook)
	fmt.Println("Listening 0.0.0.0:8085")
	log.Fatal(http.ListenAndServe(":8085", nil))
}

func SetPatliteMode(mode [7]int) {
	data := 0
	for i := 0; i < 7; i++ {
		data |= (mode[i] & 0x1) << (6 - i)
	}
	conn, err := net.Dial("udp", "172.16.254.240:10000")
	if err != nil {
		log.Println("UDP send error:", err)
		return
	}
	defer conn.Close()
	fmt.Fprintf(conn, "%c%c", 0x57, data)
}
