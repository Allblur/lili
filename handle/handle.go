package handle

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Account struct {
	Name string
	Key  string
	Cx   string
}

type Items struct {
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
	Items []Items
}

type Pag struct {
	Num  int8
	Cls  string
	Q    string
	IsOn bool
}

type Result struct {
	Items    []Items
	Start    string
	Q        string
	HasItems bool
	Pag      []Pag
}

type GptApiMessages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type GptApiQueryParams struct {
	Messages    []GptApiMessages `json:"messages"`
	Model       string           `json:"model"`
	Temperature float32          `json:"temperature"`
	Stream      bool             `json:"stream"`
}

type ChatCompletionStreamChoiceDelta struct {
	Content string `json:"content"`
}

type ChatCompletionStreamChoice struct {
	Index        int                             `json:"index"`
	Delta        ChatCompletionStreamChoiceDelta `json:"delta"`
	FinishReason string                          `json:"finish_reason"`
}

type ChatCompletionStreamResponse struct {
	ID      string                       `json:"id"`
	Object  string                       `json:"object"`
	Created int64                        `json:"created"`
	Model   string                       `json:"model"`
	Choices []ChatCompletionStreamChoice `json:"choices"`
}

type ApiParams struct {
	Key string `json:"key"`
	GptApiQueryParams
}

func Index(w http.ResponseWriter, r *http.Request) {
	var apiParams ApiParams
	fmt.Println(apiParams.Model)
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
	var accounts []Account
	var pag []Pag
	origin := "https://www.googleapis.com/customsearch/v1"
	q := r.URL.Query().Get("q")
	start := r.URL.Query().Get("start")
	a := os.Getenv("ACCOUNT")
	data := &Result{
		Items:    []Items{},
		Q:        q,
		Start:    "1",
		HasItems: false,
		Pag:      []Pag{},
	}
	if q == "" {
		q = r.FormValue("q")
	}
	if start == "" {
		s := r.FormValue("start")
		if s == "" {
			s = "1"
		}
		start = s
	}
	arr := []int8{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	index, _ := strconv.Atoi(start)
	for _, v := range arr {
		cls := ""
		ison := false
		if index == int(v) {
			cls = "on"
			ison = true
		}
		pag = append(pag, Pag{Num: v, Cls: cls, Q: q, IsOn: ison})
	}
	if q == "" || a == "" {
		generateHTML(w, data, []string{"searchlayout", "search"})
		return
	}
	json.Unmarshal([]byte(a), &accounts)
	i := rand.Int31n(int32(len(accounts)))
	fmt.Println(accounts[i].Key)
	fmt.Println(accounts[i].Cx)
	str := strings.ReplaceAll(q, " ", "")
	url := fmt.Sprintf("%s?q=%s&key=%s&cx=%s&num=%d", origin, str, accounts[i].Key, accounts[i].Cx, 10)
	if start != "" {
		url = fmt.Sprintf("%s?q=%s&key=%s&cx=%s&start=%d&num=%d", origin, str, accounts[i].Key, accounts[i].Cx, (index-1)*10+1, 10)
	}
	fmt.Println("url：" + url)
	b, err := fetch(url)
	if err != nil {
		generateHTML(w, data, []string{"searchlayout", "search"})
		return
	}
	var searchResult *SearchResult
	json.Unmarshal(b, &searchResult)
	hasItems := searchResult.Items != nil && len(searchResult.Items) > 0
	s := start
	if hasItems && searchResult.Queries.NextPage != nil && len(searchResult.Queries.NextPage) > 0 {
		s = strconv.Itoa(searchResult.Queries.NextPage[0].StartIndex)
	}
	data.Items = searchResult.Items
	data.Start = s
	data.HasItems = hasItems
	data.Pag = pag
	fmt.Println("fetch end")
	generateHTML(w, data, []string{"searchlayout", "search"})
}

func Stream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("access-control-allow-headers", "authorization, Content-Type")
	w.Header().Set("access-control-allow-methods", "*")
	w.Header().Set("access-control-allow-origin", "*")
	/* w.Header().Set("cross-origin-embedder-policy", "require-corp")
	w.Header().Set("cross-origin-opener-policy", "same-origin")
	w.Header().Set("cross-origin-resource-policy", "same-origin") */
	key := os.Getenv("OPENAI_API_KEY")
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Params Don't null")
		return
	}
	defer r.Body.Close()
	var apiParams ApiParams
	err = json.Unmarshal(body, &apiParams)
	if err != nil {
		fmt.Println(err)
		// w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "Invalid JSON format")
		return
	}
	fmt.Printf("key=%s\n OPEN KEY=%s", apiParams.Key, key)
	if apiParams.Key == "" {
		apiParams.Key = key
	}
	if apiParams.Model == "" {
		apiParams.Model = "gpt-3.5-turbo"
	}
	if apiParams.Temperature == 0 {
		apiParams.Temperature = 0.75
	}
	requestBody := GptApiQueryParams{
		Stream:      true,
		Model:       apiParams.Model,
		Temperature: apiParams.Temperature,
		Messages:    apiParams.Messages,
	}
	requestBodyBytes, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format, api")
		return
	}
	// fmt.Println(string(requestBodyBytes))
	fmt.Println("User：" + apiParams.Messages[len(apiParams.Messages)-1].Content)
	req, err := http.NewRequest(http.MethodPost,
		"https://api.openai.com/v1/chat/completions",
		bytes.NewBuffer(requestBodyBytes))

	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Network error")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", apiParams.Key))
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Network error, api")
		return
	}
	defer resp.Body.Close()
	str := strings.Builder{}
	// 处理stream结果
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		var chatCompletionStream ChatCompletionStreamResponse
		headerData := "data: "
		line := strings.TrimSpace(scanner.Text())
		// fmt.Println(line + "\n")
		if strings.HasPrefix(line, headerData) && line != "data: [DONE]" {
			// line = strings.TrimPrefix(line, headerData)
			err = json.Unmarshal([]byte(line[len(headerData):]), &chatCompletionStream)
			if err == nil && chatCompletionStream.Choices != nil && chatCompletionStream.Choices[0].FinishReason != "stop" {
				content := chatCompletionStream.Choices[0].Delta.Content
				str.WriteString(content)
				w.Write([]byte(content))
				w.(http.Flusher).Flush()
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "read stream failed.")
		return
	}
	fmt.Println("\nAI：" + str.String())
}

func unescaped(x string) any {
	return template.HTML(x)
}

func generateHTML(w http.ResponseWriter, data any, fileNames []string) {
	var files []string
	t := template.New("")
	t = t.Funcs(template.FuncMap{"unescaped": unescaped})
	for _, file := range fileNames {
		files = append(files, fmt.Sprintf("templates/%s.html", file))
	}
	templates := template.Must(t.ParseFiles(files...))
	templates.ExecuteTemplate(w, "layout", data)
}

func fetch(url string) ([]byte, error) {
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
	resp, err := http.DefaultClient.Do(request)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}
	byte, _ := ioutil.ReadAll(resp.Body)
	return byte, nil
}
