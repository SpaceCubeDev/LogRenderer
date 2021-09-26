package main

import (
	"encoding/json"
	"fmt"
)

const (
	eventAdd   = "ADD"
	eventReset = "RESET"
)

type fileEvent struct {
	eventType string
	content   string
}

func (evt fileEvent) String() string {
	return fmt.Sprintf("Type: %s / Content: %s\n", evt.eventType, evt.content)
}

type Event struct {
	Type   string `json:"type"`
	Server string `json:"server"`
	// ServerDisplayName string `json:"server_display_name"`
	Content string `json:"content"`
}

func (event Event) String() string {
	var str string
	str += "Type: " + event.Type + "\n"
	str += "Server: " + event.Server + "\n"
	// str += "DisplayName: " + event.ServerDisplayName + "\n"
	switch event.Type {
	case eventAdd:
		str += "Content: " + event.Content + "\n"
	default:
		str += "/!\\ Unknown event type ! /!\\\n"
	}
	return str
}

// Json returns the json version of the Event
func (event Event) Json() []byte {
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		printError(err)
		return []byte(err.Error())
	}
	return jsonBytes
}
