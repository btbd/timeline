package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
)

type Post struct {
	From     string `json:"from"`
	Message  string `json:"message"`
	Image    string `json:"image"`
	Date     int64  `json:"date"`
	Id       int64  `json:"id"`
	
	Raw      []byte `json:"-"`
	RawType  string `json:"-"`
}

var posts []Post = []Post{}
var tokens_path string
var verbose bool
var header string
var global_id int64

func Print(format string, args ...interface{}) {
	if verbose {
		fmt.Printf(format, args...)
	}
}

func CheckPostVar(j map[string]interface{}, v string) string {
	e, ok := j[v]

	if !ok || e == nil {
		return `"` + v + `" must be defined`
	}

	if reflect.TypeOf(e).String() != "string" {
		return `"` + v + `" must be a string`
	}

	if len(e.(string)) == 0 {
		return `"` + v + `" cannot be an empty string`
	}

	return ""
}

func BadRequest(w http.ResponseWriter, err string) {
	w.WriteHeader(http.StatusBadRequest)
	w.Write([]byte(err))
}

func HandleFileRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	bytes, err := ioutil.ReadFile("index.html")

	if err == nil {
		if len(header) != 0 {
			bytes = []byte(strings.Replace(string(bytes), "***Timeline***", header, -1))
		}

		w.Write(bytes)
	} else {
		w.Write([]byte("404: \"" + r.URL.Path[1:] + "\" not found\n"))
	}
}

func HandlePost(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method == "POST" {
		if len(tokens_path) != 0 {
			bytes, err := ioutil.ReadFile(tokens_path)
			if err == nil {
				auth := r.Header.Get("Authorization")
				if len(auth) < 8 || !strings.HasPrefix(auth, "Bearer ") {
					BadRequest(w, "No Bearer token provided")
					return
				}

				auth = auth[7:len(auth)]

				valid := false
				for _, token := range strings.Split(string(bytes), "\n") {
					if i := strings.Index(token, "#"); i != -1 {
						token = token[0:i]
					}
					token = strings.Trim(token, " \r")

					if auth == token {
						valid = true
						break
					}
				}

				if !valid {
					w.WriteHeader(http.StatusUnauthorized)
					w.Write([]byte("Incorrect token provided"))
					return
				}
			} else {
				fmt.Fprintf(os.Stderr, "Failed to open token file \"%v\"\n", tokens_path)
				os.Exit(1)
			}
		}

		mediaType, params, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
		if err != nil {
			BadRequest(w, err.Error())
			return
		}

		if strings.HasPrefix(mediaType, "multipart/") {
			mr := multipart.NewReader(r.Body, params["boundary"])

			p, err := mr.NextPart()
			if err == io.EOF {
				BadRequest(w, "No JSON part detected")
				return
			} else if err != nil {
				BadRequest(w, err.Error())
				return
			}

			if !strings.Contains(p.Header.Get("Content-Type"), "json") {
				BadRequest(w, "First part must be JSON")
				return
			}

			body, err := ioutil.ReadAll(p)
			if err != nil {
				BadRequest(w, err.Error())
				return
			}

			j := make(map[string]interface{})

			err = json.Unmarshal(body, &j)
			if err != nil {
				BadRequest(w, "JSON parse error: "+err.Error())
				return
			}

			for _, v := range []string{"from", "message"} {
				if e := CheckPostVar(j, v); e != "" {
					BadRequest(w, e)
					return
				}
			}

			p, err = mr.NextPart()
			if err == io.EOF {
				BadRequest(w, "No image part detected")
				return
			} else if err != nil {
				BadRequest(w, err.Error())
				return
			}

			if !strings.HasPrefix(p.Header.Get("Content-Type"), "image/") {
				BadRequest(w, "Second part must be an image")
				return
			}

			body, err = ioutil.ReadAll(p)
			if err != nil {
				BadRequest(w, err.Error())
				return
			}

			if len(body) == 0 {
				BadRequest(w, "The image cannot be null")
				return
			}

			var post Post
			post.From = j["from"].(string)
			post.Message = j["message"].(string)
			post.Raw = body
			post.RawType = p.Header.Get("Content-Type")
			post.Date = time.Now().UTC().Unix()
			
			post.Id = global_id
			global_id++
			
			post.Image = "./image?id=" + strconv.FormatInt(post.Id, 10)

			Print("%v - \"%v\"\n", time.Unix(post.Date, 0).UTC(), post.From)

			posts = append(posts, post)
			if len(posts) > 100 {
				posts = posts[1:101]
			}
		}
	} else {
		BadRequest(w, "Expected multipart POST request with 1st part as JSON containing 'from' and 'message', and 2nd part containing an image to post")
	}
}

func HandleImage(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	
	if id, err := strconv.ParseInt(r.FormValue("id"), 10, 64); err == nil && id > 0 {
		for _, post := range posts {
			if post.Id == id {
				w.Header().Set("Content-Type", post.RawType)
				w.Write(post.Raw)
				return
			}
		}
	}
	
	BadRequest(w, "Could not find image")
}

func HandleTimeline(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	w.Header().Set("Content-Type", "application/json")

	var new_posts []Post = []Post{}

	if id, err := strconv.ParseInt(r.FormValue("id"), 10, 64); err == nil {
		for i := len(posts) - 1; i > -1; i-- {
			if posts[i].Id <= id {
				break
			}
			
			new_posts = append(new_posts, posts[i])
		}
	} else {
		new_posts = posts
	}

	bytes, err := json.Marshal(new_posts)
	if err != nil {
		return
	}

	w.Write(bytes)
}

func main() {
	global_id = time.Now().UTC().Unix()
	
	header = "Timeline"
	tokens_path = "tokens"
	port := 80
	crt := ""
	key := ""

	usage := flag.Usage
	flag.Usage = func() {
		fmt.Println("Timeline")
		usage()
	}

	flag.IntVar(&port, "p", port, "port")
	flag.BoolVar(&verbose, "v", verbose, "display debug info")
	flag.StringVar(&header, "h", header, "header title for web page")
	flag.StringVar(&crt, "crt", crt, "certificate for TLS")
	flag.StringVar(&key, "key", key, "key for TLS")
	flag.StringVar(&tokens_path, "tokens", tokens_path, "tokens for authenticating requests")

	flag.Parse()
	
	if len(tokens_path) != 0 {
		_, err := ioutil.ReadFile(tokens_path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to open token file \"%v\"\n", tokens_path)
			os.Exit(1)
		}
	}

	http.HandleFunc("/", HandleFileRequest)
	http.HandleFunc("/post", HandlePost)
	http.HandleFunc("/image", HandleImage)
	http.HandleFunc("/timeline", HandleTimeline)

	if len(crt) > 0 && len(key) > 0 {
		if err := http.ListenAndServeTLS(":"+strconv.Itoa(port), crt, key, nil); err != nil {
			fmt.Fprintf(os.Stderr, "(HTTPS) Error listening on port %d:\n\t%s\n", port, err)
			os.Exit(1)
		}
	} else if err := http.ListenAndServe(":"+strconv.Itoa(port), nil); err != nil {
		fmt.Fprintf(os.Stderr, "(HTTP) Error listening on port %d:\n\t%s\n", port, err)
		os.Exit(1)
	}
}
