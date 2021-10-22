package dc_test

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"testing"

	"github.com/toriato/dc"
)

var (
	testdata struct {
		Gallog struct {
			Guestbook struct {
				Credentials dc.Credentials
			} `json:"guestbook"`
		} `json:"gallog"`

		Session struct {
			Login struct {
				Invalid        dc.Credentials `json:"invalid"`
				InvalidTOTPKey dc.Credentials `json:"invalidTOTPKey"`
				Valid          struct {
					dc.Credentials
					Nickname string `json:"nickname"`
				} `json:"valid"`
				ValidTOTP struct {
					dc.Credentials
					Nickname string `json:"nickname"`
				} `json:"validTOTPKey"`
			} `json:"login"`
		} `json:"session"`

		User struct {
			Credentials []struct {
				dc.Credentials
				Nickname string   `json:"nickname"`
				Flags    []string `json:"flags"`
			} `json:"credentials"`
		} `json:"user"`
	}
)

func TestMain(m *testing.M) {
	raw, err := ioutil.ReadFile("testdata/testdata.json")
	if err != nil {
		log.Fatal(err)
	}

	if err := json.Unmarshal(raw, &testdata); err != nil {
		log.Fatal(err)
	}

	code := m.Run()
	os.Exit(code)
}
