package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"html"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
)

var port = flag.Uint("p", 2001, "the port to listen on")

func main() {
	flag.Parse()

	gb := GuestBook{}

	if len(flag.Args()) != 1 {
		fmt.Println("must provide only a file name")
		os.Exit(1)
	} else {
		gb.File = flag.Args()[0]
	}


	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			gb.handleGet(w, r)
		case http.MethodPost:
			gb.handlePost(w, r)
		default:
			http.Error(w,
				"method not allowed",
				http.StatusMethodNotAllowed)
		}
	})

	http.ListenAndServe(fmt.Sprintf(":%d", *port), nil)
}

func (gb *GuestBook) handleGet(w http.ResponseWriter, r *http.Request) {
	byt, err := gb.MarshalJSON()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(byt)
}

func (gb *GuestBook) handlePost(w http.ResponseWriter, r *http.Request) {
	buf := make([]byte, 1024) // max msg length
	n, err := r.Body.Read(buf)
	r.Body.Close()
	if err != nil {
		http.Error(w, err.Error(), 400)
	}

	gb.Write(buf[:n])
	gb.handleGet(w, r)
}

// GuestBook is the name of a file that is lazily read
// read and parsed
type GuestBook struct {
	File string
	lock sync.Mutex
}

func (gb *GuestBook) MarshalJSON() ([]byte, error) {
	arr, err := gb.Entries()
	if err != nil {
		return nil, err
	}
	return json.Marshal(arr)
}

func (gb *GuestBook) Entries() ([]string, error) {
	gb.lock.Lock()

	byt, err := ioutil.ReadFile(gb.File)
	if err != nil {
		gb.lock.Unlock()
		return nil, err
	}
	gb.lock.Unlock()


	arr := bytes.Split(byt, []byte{'\n'})
	ret := make([]string, len(arr))
	enc := base64.StdEncoding

	for i, v := range arr {
		buf := make([]byte, enc.DecodedLen(len(v)))
		enc.Decode(buf, v)
		ret[i] = html.EscapeString(string(buf))
	}

	return ret, nil
}

func (gb *GuestBook) Write(data []byte) error {
	gb.lock.Lock()
	defer gb.lock.Unlock()

	file, err := os.OpenFile(gb.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	w := base64.NewEncoder(base64.StdEncoding, file)
	w.Write(data)
	w.Close()
	file.Write([]byte{'\n'})
	file.Close()
	return nil
}
