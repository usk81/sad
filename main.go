package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"time"
)

const (
	count   = 1000
	baseURL = "https://slack.com/api"
)

type listResponse struct {
	OK    bool `json:"ok"`
	Files []struct {
		ID string `json:"id"`
	} `json:"files"`
	Error string `json:"error"`
}

type deleteResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error"`
}

func main() {
	var from, to int
	var token string
	flag.IntVar(&from, "from", 0, "Filter files created after this timestamp")
	flag.IntVar(&to, "to", int(time.Now().AddDate(-1, -6, 0).Unix()), "Filter files created before this timestamp. if request zero, set current timestamp")
	flag.StringVar(&token, "token", os.Getenv("SLACK_ACCESS_TOKEN"), "slack access token")

	flag.Parse()

	if err := destroy(token, from, to); err != nil {
		fmt.Println(fmt.Errorf("fail to delete slack attachements : %w", err))
	}
	fmt.Println("finish")
}

func destroy(token string, from, to int) (err error) {
	if token == "" {
		return fmt.Errorf("token is empty")
	}

	i := 0
	for {
		l, err := list(token, from, to)
		if err != nil {
			return err
		}
		i++

		if i >= 30 {
			time.Sleep(5 * time.Second)
			i = 0
		}

		if len(l.Files) == 0 {
			break
		}

		for _, f := range l.Files {
			fmt.Printf("delete: %s\n", f.ID)
			if err = delete(token, f.ID); err != nil {
				return err
			}
			i++
			if i >= 30 {
				time.Sleep(5 * time.Second)
				i = 0
			}
		}
	}
	return nil
}

func list(token string, from, to int) (result listResponse, err error) {
	q := url.Values{}
	q.Set("token", token)
	q.Set("show_files_hidden_by_limit", "true")
	q.Set("count", strconv.Itoa(count))
	if from != 0 {
		q.Set("ts_from", strconv.Itoa(from))
	}
	if to != 0 {
		q.Set("ts_to", strconv.Itoa(to))
	}
	err = request(http.MethodGet, "files.list", q, &result)
	return result, err
}

func delete(token string, id string) (err error) {
	q := url.Values{}
	q.Set("token", token)
	q.Set("file", id)

	var resp deleteResponse
	if err = request(http.MethodPost, "files.delete", q, &resp); err != nil {
		return err
	}
	if resp.Error != "" {
		return fmt.Errorf(resp.Error)
	}
	return nil
}

func request(method, p string, q url.Values, result interface{}) (err error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	u.Path = path.Join(u.Path, p)
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(method, u.String(), nil)
	if err != nil {
		return
	}
	client := http.DefaultClient
	resp, err := client.Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not return status ok: status: %d", resp.StatusCode)
	}

	if result != nil {
		if err = json.NewDecoder(resp.Body).Decode(result); err != nil {
			return
		}
	}
	return nil
}
