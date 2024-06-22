package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type Ticket struct {
	EventID int32  `json:"event_id"`
	Name    string `json:"name"`
}

type Event struct {
	ID       int32    `json:"id"`
	Name     string   `json:"name"`
	Capacity int32    `json:"capacity"`
	Tickets  []Ticket `json:"tickets"`
}

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

func getEvents() ([]Event, error) {
	resp, err := http.Get("http://localhost:9000/events")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var events []Event
	json.Unmarshal(response, &events)
	return events, nil
}

func postEvent(event Event) (*Event, error) {
	postBody, _ := json.Marshal(event)
	body := bytes.NewBuffer(postBody)
	resp, err := http.Post("http://localhost:9000/events", "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	response, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var e Event
	json.Unmarshal(response, &e)
	return &e, nil
}

func deleteEvent(event *Event) (int, error) {
	client := &http.Client{}
	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:9000/events/%d", event.ID), nil)
	if err != nil {
		return -1, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return -1, err
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}

func createTicket(event *Event) (*Ticket, error) {
	ticket := &Ticket{
		EventID: event.ID,
		Name:    randSeq(rand.Intn(50)),
	}
	fmt.Println("Generated ticket: ", ticket)
	postBody, _ := json.Marshal(ticket)
	body := bytes.NewBuffer(postBody)
	resp, err := http.Post(fmt.Sprintf("http://localhost:9000/events/%d/ticket", ticket.EventID), "application/json", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var res Ticket

	if resp.StatusCode == 200 {
		response, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		json.Unmarshal(response, &res)
		return &res, nil
	}

	return nil, fmt.Errorf("ERROR API: %d", resp.StatusCode)
}

func main() {
	eventChannel := make(chan *Event, 1)

	deleteChannel := make(chan *Event, 1)

	go func(events chan *Event) {
		for event := range events {
			_, err := deleteEvent(event)
			if err != nil {
				log.Printf("Could not delete event ID: %d: %s", event.ID, err.Error())
				continue
			}
			fmt.Println("Deleted event: ", event)
		}
	}(deleteChannel)

	go func(events chan *Event) {
		for event := range events {
			for {
				fmt.Println("Got event: ", event)
				ticket, _ := createTicket(event)
				if ticket == nil {
					deleteChannel <- event
					break
				}
				fmt.Println("Created ticket: ", ticket)
			}
		}
	}(eventChannel)

	go func() {
		for {
			getEvents()
			time.Sleep(time.Duration(rand.Intn(10) * int(time.Millisecond)))
		}
	}()

	for {
		event, err := postEvent(Event{
			Name:     randSeq(rand.Intn(50)),
			Capacity: rand.Int31n(500),
		})
		if err != nil {
			panic(err)
		}
		eventChannel <- event
		time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
	}
}
