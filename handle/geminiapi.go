package handle

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type Parts struct {
	Text string `json:"text"`
}

type Content struct {
	Role  string  `json:"role"`
	Parts []Parts `json:"parts"`
}

type SafetySettings struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GenerationConfig struct {
	StopSequences   []string `json:"stopSequences"`
	Temperature     float64  `json:"temperature"`
	MaxOutputTokens int      `json:"maxOutputTokens"`
	TopP            float64  `json:"topP"`
	TopK            int      `json:"topK"`
}

type RequestBody struct {
	Contents []Content `json:"contents"`
}

type ApiRequestBody struct {
	Contents         []Content        `json:"contents"`
	SafetySettings   []SafetySettings `json:"safetySettings"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

func Geminiapi(w http.ResponseWriter, r *http.Request) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-pro:streamGenerateContent?key=%s", os.Getenv("GEMINI_API_KEY"))
	reqBody := RequestBody{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format")
		return
	}
	contents := []Content{}
	for i := 0; i < len(reqBody.Contents); i++ {
		parts := []Parts{}
		for j := 0; j < len(reqBody.Contents[i].Parts); j++ {
			parts = append(parts, reqBody.Contents[i].Parts[j])
		}
		contents = append(contents, Content{
			Parts: parts,
			Role:  reqBody.Contents[i].Role,
		})
	}
	requestBody := ApiRequestBody{
		Contents: contents,
		SafetySettings: []SafetySettings{
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_ONLY_HIGH",
			},
		},
		GenerationConfig: GenerationConfig{
			StopSequences:   []string{"Title"},
			Temperature:     1.0,
			MaxOutputTokens: 800,
			TopP:            0.8,
			TopK:            10,
		},
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, "Invalid JSON format. jsonData error %v", err)
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json; charset=UTF-8")
	/* req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive") */
	// Send the request and get the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprintf(w, "resp error %v", err)
		return
	}
	defer resp.Body.Close()
	// str := strings.Builder{}
	b, _ := io.ReadAll(resp.Body)
	fmt.Printf("\nresp.Body==%s", string(b))
	// 处理stream结果
	scanner := bufio.NewScanner(resp.Body)
	scanner.Split(func(data []byte, atEOF bool) (int, []byte, error) {
		if i := bytes.Index(data, []byte("}\n,\r\n")); i >= 0 {
			return i + 5, data[0 : i+1], nil
		}
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	})
	for {
		if !scanner.Scan() {
			fmt.Fprint(w, "read response stream failed.")
			return
		}
		if err = scanner.Err(); err != nil {
			fmt.Println(err)
			fmt.Fprint(w, "read response stream failed. err...")
			return
		}
		txt := scanner.Text()
		// remove head '[' and tail ']'
		txt = strings.TrimLeft(txt, "[,\r\n")
		txt = strings.TrimRight(txt, "],\r\n")
		fmt.Println(txt)
		w.Write([]byte(txt))
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()
	}
	/* for scanner.Scan() {
		// str.WriteString(scanner.Text())
		fmt.Println(scanner.Text())
		w.Write([]byte(scanner.Text()))
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()
	} */

	// fmt.Println("\nAI end." /*  + str.String() */)
}
