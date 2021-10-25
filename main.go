package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"bufio"
	"net/http"
	"os"
	"sync"
	"html/template"
	_ "embed"
)

var port = flag.Uint("p", 2001, "the port to listen on")


//go:embed index.html.tmpl
var indexTmplStr string

var indexTmpl = template.Must(template.New("index").Parse(indexTmplStr))

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
	arr, err := gb.Entries()
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	err = indexTmpl.Execute(w, arr)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
}

func (gb *GuestBook) handlePost(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1024)
	err := r.ParseForm()
	if err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	gb.AddSignature(r.PostForm.Get("signature"))

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
	gb.lock.Unlock()

	if err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(bytes.NewReader(byt))
	ret := make([]string, 0, 1024)
	enc := base64.StdEncoding
	for scanner.Scan() {
		decoded, err := enc.DecodeString(scanner.Text())
		if err != nil {
			return nil, err
		}
		ret = append(ret, string(decoded))
	}

	for i := 0; i < len(ret) / 2; i++ {
		ret[i], ret[len(ret) - i - 1] = ret[len(ret) - i - 1], ret[i]
	}

	return ret, nil
}

func (gb *GuestBook) AddSignature(data string) error {
	if len(data) == 0 {
		return nil
	}
	gb.lock.Lock()
	defer gb.lock.Unlock()

	file, err := os.OpenFile(gb.File, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}

	w := base64.NewEncoder(base64.StdEncoding, file)
	w.Write([]byte(data))
	w.Close()
	file.Write([]byte{'\n'})
	file.Close()
	return nil
}
