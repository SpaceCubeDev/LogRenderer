package main

import (
	"encoding/json"
	"fmt"
)

const (
	eventAdd   = "ADD"
	eventReset = "RESET"
	eventError = "ERROR"
)

type fileEvent struct {
	eventType string
	content   string
}

func (evt fileEvent) String() string {
	return fmt.Sprintf("Type: %s / Content: %s\n", evt.eventType, evt.content)
}

type Event struct {
	Type      string `json:"type"`
	Server    string `json:"server"`
	isDynamic bool
	instance  string
	// ServerDisplayName string `json:"server_display_name"`
	Content string `json:"content"`
	Message string `json:"message"`
}

func (event Event) String() string {
	str := "Type: " + event.Type + "\n"

	switch {
	case event.Type == eventAdd || event.Type == eventReset:
		str += "Server: " + event.Server + "\n"
		if event.isDynamic {
			str += "Instance: " + event.instance + "\n"
		}
		fallthrough
	case event.Type == eventAdd:
		str += "Content: " + event.Content + "\n"
	case event.Type == eventError:
		str += "Message: " + event.Message + "\n"
	default:
		str += "/!\\ Unknown event type ! /!\\\n"
	}

	/*if event.Type != eventAdd && event.Type != eventReset && event.Type != eventError {
		return str + "/!\\ Unknown event type ! /!\\\n"
	}
	if event.Type == eventError {
		return str + "Message: " + event.Message + "\n"
	}
	str += "Server: " + event.Server + "\n"
	// str += "DisplayName: " + event.ServerDisplayName + "\n"
	if event.Type == eventAdd {
		str += "Content: " + event.Content + "\n"
	}*/
	return str
}

// Json returns the json version of the Event
func (event Event) Json() []byte {
	if event.isDynamic {
		event.Server = joinWSServer(event.Server, event.instance)
	}
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		printError(fmt.Errorf("failed to marshal event: %w", err))
		return []byte("{}")
	}
	return jsonBytes
}
