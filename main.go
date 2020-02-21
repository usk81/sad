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
	Users []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Deleted bool   `json:deleted`
		Updated int    `json:updated`
	} `json:"members"`
	Channels []struct {
		ID      string `json:"id"`
		Created int    `json:"created"`
	} `json:"channels"`
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

	// ru, err := userList(token)
	// if err != nil {
	// 	return err
	// }
	// us := ru.Users

	// rc, err := channelList(token)
	// if err != nil {
	// 	return err
	// }
	// cs := rc.Channels

	// fmt.Printf("%v\n", us)

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

	// for _, u := range us {
	// 	fmt.Printf("user: %s\n", u.Name)
	// 	for _, ch := range cs {
	// 		if ch.Created > to {
	// 			continue
	// 		}
	// 		if u.Deleted && ch.Created > u.Updated {
	// 			continue
	// 		}
	// 		for {
	// 			l, err := list(token, u.ID, ch.ID, from, to)
	// 			if err != nil {
	// 				return err
	// 			}
	// 			i++

	// 			if i >= 30 {
	// 				time.Sleep(10 * time.Second)
	// 				i = 0
	// 			}

	// 			if len(l.Files) == 0 {
	// 				break
	// 			}

	// 			for _, f := range l.Files {
	// 				fmt.Printf("delete: %s\n", f.ID)
	// 				if err = delete(token, f.ID); err != nil {
	// 					return err
	// 				}
	// 				i++
	// 				if i >= 30 {
	// 					time.Sleep(10 * time.Second)
	// 					i = 0
	// 				}
	// 			}
	// 		}
	// 	}
	// }
	return nil
}

// func list(token, u, ch string, from, to int) (result listResponse, err error) {
func list(token string, from, to int) (result listResponse, err error) {
	q := url.Values{}
	q.Set("token", token)
	// q.Set("user", u)
	// q.Set("channel", ch)
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

func userList(token string) (result listResponse, err error) {
	q := url.Values{}
	q.Set("token", token)
	err = request(http.MethodGet, "users.list", q, &result)
	return
}

func channelList(token string) (result listResponse, err error) {
	q := url.Values{}
	q.Set("token", token)
	err = request(http.MethodGet, "channels.list", q, &result)
	return
}

func request(method, p string, q url.Values, result interface{}) (err error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return
	}
	u.Path = path.Join(u.Path, p)
	u.RawQuery = q.Encode()

	// fmt.Println(u.String())

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
