package handle

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
)

type engin struct {
	Name string
	Key  string
	Cx   string
}

type SearchResult struct {
	Queries struct {
		NextPage []struct {
			Title        string
			TotalResults string
			SearchTerms  string
			Count        int64
			StartIndex   int
		}
	}
	SearchInformation struct {
		SearchTime            float64
		FormattedSearchTime   string
		TotalResults          string
		FormattedTotalResults string
	}
	Items []struct {
		Title       string
		HtmlTitle   string
		Link        string
		DisplayLink string
		Snippet     string
		HtmlSnippet string
		Pagemap     struct {
			CseThumbnail []struct {
				Width  string
				Height string
				Src    string
			}
			Imageobject []struct {
				Width  string
				Height string
				Url    string
			}
			Answer []struct {
				Upvotecount  string
				Commentcount string
				Datemodified string
				Datecreated  string
				Text         string
				Url          string
			}
			Person []struct {
				Image string
				Name  string
				Url   string
			}
		}
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.Cookies())
	for _, cookie := range r.Cookies() {
		fmt.Println("cookie name == " + cookie.Name + "\ncookie value == " + cookie.Value)
	}
	var data = struct {
		Name string
	}{
		Name: "golang template parse",
	}
	files := []string{"layout", "index"}
	generateHTML(w, data, files)
}

func Search(w http.ResponseWriter, r *http.Request) {
	var e []engin
	origin := "https://www.googleapis.com/customsearch/v1"
	q := r.URL.Query().Get("q")
	n := r.URL.Query().Get("n")
	start := r.URL.Query().Get("start")
	i := rand.Int31n(5)
	engins := os.Getenv("engins")
	fmt.Println("q=" + q)
	fmt.Println("n=" + n)
	fmt.Println("engins=" + engins)
	json.Unmarshal([]byte(engins), &e)
	fmt.Println(e)
	url := fmt.Sprintf("%s?q=%s&key=%s&cx=%s&num=%s", origin, q, e[i].Key, e[i].Cx, n)
	if start != "" {
		url = fmt.Sprintf("%s?q=%s&key=%s&cx=%s&start=%s&num=%s", origin, q, e[i].Key, e[i].Cx, start, n)
	}
	b := fetch(url)
	generateHTML(w, b, []string{"layout", "search"})
}

func generateHTML(w http.ResponseWriter, data any, fileNames []string) {
	var files []string
	for _, file := range fileNames {
		files = append(files, fmt.Sprintf("templates/%s.html", file))
	}
	templates := template.Must(template.ParseFiles(files...))
	// fmt.Fprintf(w, "hello go")
	templates.ExecuteTemplate(w, "layout", data)
}

func fetch(url string) []byte {
	var headers = map[string]string{
		"Accept":          "*/*",
		"Accept-Language": "zh-CN,zh;q=0.8,gl;q=0.6,zh-TW;q=0.4;en",
		"Connection":      "keep-alive",
		"Host":            "www.googleapis.com",
		"User-Agent":      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/111.0.0.0 Safari/537.36 Edg/111.0.1661.62",
		"Origin":          "https://www.googleapis.com",
		"Referer":         "https://www.googleapis.com",
	}

	request, _ := http.NewRequest(http.MethodGet, url, nil)
	for k, v := range headers {
		request.Header.Add(k, v)
	}
	resp, _ := http.DefaultClient.Do(request)
	if resp != nil {
		defer resp.Body.Close()
	}
	byte, _ := ioutil.ReadAll(resp.Body)
	return byte
}
