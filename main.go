package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

// secret.go
var (
	discordWebhookUrl = DISCORD_WEBHOOK_URL
	discordUserId     = DISCORD_USER_ID
)

func main() {
	event, err := formatEvent()
	if err != nil {
		event = fmt.Sprintf("### ERROR\n```\n%v\n\nargs: %s```", err, strings.Join(os.Args, " "))
	}
	for i := 0; ; i++ {
		err := discord(discordWebhookUrl, discordUserId, event)
		if err == nil || i == 3 {
			break
		}
		fmt.Println(err)
		fmt.Println("Retrying", i)
		time.Sleep(time.Second * time.Duration(i))
	}
}

func formatEvent() (string, error) {
	title, err := getArg(1)
	if err != nil {
		return "", err
	}
	logName, err := getArg(2)
	if err != nil {
		return "", err
	}
	eventId, err := getArg(3)
	if err != nil {
		return "", err
	}

	event, err := getEvent(logName, eventId)
	if err != nil {
		return "", err
	}
	var unix int64
	_, err = fmt.Sscanf(event["TimeCreated"].(string), "/Date(%d)/", &unix)
	if err != nil {
		return "", err
	}
	createdAt := time.Unix(unix/1000, 0)
	return fmt.Sprintf("### %s\n`event`: %s:%s\n`time`: %s\n`message`: %s", title, logName, eventId, createdAt.String(), event["Message"]), nil
}

func getArg(n int) (string, error) {
	if len(os.Args) <= n {
		return "", fmt.Errorf("no argument: %d", n)
	}
	return os.Args[n], nil
}

func getEvent(logName, eventId string) (map[string]any, error) {
	args := []string{
		"-nologo",
		"-noprofile",
		"get-winevent",
		"-maxevents",
		"1",
		"-filterhashtable",
		fmt.Sprintf(`@{Id=%s;LogName="%s"}`, eventId, logName),
		"|",
		"ConvertTo-Json",
	}
	out, err := exec.Command("powershell", args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s: %w", string(out), err)
	}
	var data map[string]any
	return data, json.Unmarshal(out, &data)
}

func discord(url, uid string, message string) error {
	body := map[string]any{
		"content": fmt.Sprintf("%s\n-# <@%s>", message, uid),
	}
	buf := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(body)
	if err != nil {
		return err
	}
	data := buf.Bytes()
	fmt.Print(string(data))
	res, err := http.Post(
		url+"?wait=true",
		"application/json",
		bytes.NewBuffer(data),
	)
	if err != nil {
		return err
	}
	resb, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != 200 && res.StatusCode != 201 {
		return errors.New(string(resb))
	}
	fmt.Println(res.Status)
	return nil
}
