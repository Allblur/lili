package handle

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Parts struct {
	Text string `json:"text"`
}

type Content struct {
	Role  string  `json:"role,omitempty"`
	Parts []Parts `json:"parts"`
}

type SafetySettings struct {
	Category  string `json:"category"`
	Threshold string `json:"threshold"`
}

type GenerationConfig struct {
	StopSequences   []string `json:"stopSequences,omitempty"`
	Temperature     float64  `json:"temperature"`
	MaxOutputTokens int      `json:"maxOutputTokens"`
	TopP            float64  `json:"topP"`
	TopK            int      `json:"topK"`
}

type ComonBody struct {
	Contents         []Content        `json:"contents"`
	GenerationConfig GenerationConfig `json:"generationConfig"`
}

type RequestBody struct {
	ComonBody
	Version string `json:"version"`
}

type ApiRequestBody struct {
	ComonBody
	SafetySettings []SafetySettings `json:"safetySettings"`
}

type GenerateContentResponse struct {
	Candidates     []GenerateContentCandidate `json:"candidates"`
	PromptFeedback PromptFeedback             `json:"promptFeedback"`
}

type GenerateContentCandidate struct {
	Content       Content        `json:"content"`
	FinishReason  string         `json:"finishReason"`
	Index         int            `json:"index"`
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}

type PromptFeedback struct {
	SafetyRatings []SafetyRating `json:"safetyRatings"`
}

type SafetyRating struct {
	Category    string `json:"category"`
	Probability string `json:"probability"`
}
type VisionReqParts struct {
	Text       string     `json:"text,omitempty"`
	InlineData InlineData `json:"inline_data,omitempty"`
}

type VisionReqContents struct {
	Parts []VisionReqParts `json:"parts"`
}

type InlineData struct {
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type VisionReq struct {
	Contents []VisionReqContents `json:"contents"`
	Version  string              `json:"version"`
}

type VisionApiReq struct {
	Contents         []VisionReqContents `json:"contents"`
	GenerationConfig GenerationConfig    `json:"generationConfig"`
	SafetySettings   []SafetySettings    `json:"safetySettings"`
}

func Geminiapi(w http.ResponseWriter, r *http.Request) {
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
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/gemini-pro:streamGenerateContent?key=%s", reqBody.Version, os.Getenv("GEMINI_API_KEY"))
	apiRequestBody := ApiRequestBody{
		ComonBody: ComonBody{
			Contents:         contents,
			GenerationConfig: reqBody.GenerationConfig,
		},
		SafetySettings: []SafetySettings{
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_ONLY_HIGH",
			},
		},
	}

	jsonData, err := json.Marshal(apiRequestBody)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format.")
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return
	}
	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	// Send the request and get the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "resp error.")
		return
	}
	defer resp.Body.Close()
	str := strings.Builder{}
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
	for scanner.Scan() {
		str.WriteString(scanner.Text())
		txt := scanner.Text()
		txt = strings.TrimLeft(txt, "[,\r\n")
		txt = strings.TrimRight(txt, "],\r\n")
		var res GenerateContentResponse
		err = json.Unmarshal([]byte(txt), &res)
		if err != nil {
			break
		}
		w.Write([]byte("[model]: " + res.Candidates[0].Content.Parts[0].Text))
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "read stream failed.")
		return
	}
	fmt.Println("\nAI end." + str.String())
}

func Geminivision(w http.ResponseWriter, r *http.Request) {
	reqBody := VisionReq{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format")
		return
	}
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/%s/models/gemini-pro-vision:streamGenerateContent?key=%s", reqBody.Version, os.Getenv("GEMINI_API_KEY"))
	apiRequestBody := VisionApiReq{
		Contents: reqBody.Contents,
		GenerationConfig: GenerationConfig{
			StopSequences:   []string{"Title"},
			Temperature:     1.0,
			MaxOutputTokens: 4096,
			TopP:            0.8,
			TopK:            10,
		},
		SafetySettings: []SafetySettings{
			{
				Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
				Threshold: "BLOCK_ONLY_HIGH",
			},
		},
	}
	jsonData, err := json.Marshal(apiRequestBody)
	// fmt.Println("jsonData==", string(jsonData))
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format.")
		return
	}

	// Create a new HTTP request
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println(err)
		return
	}

	// Set the Content-Type header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")
	// Send the request and get the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "resp error.")
		return
	}
	defer resp.Body.Close()
	str := strings.Builder{}
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
	for scanner.Scan() {
		str.WriteString(scanner.Text())
		txt := scanner.Text()
		txt = strings.TrimLeft(txt, "[,\r\n")
		txt = strings.TrimRight(txt, "],\r\n")
		var res GenerateContentResponse
		err = json.Unmarshal([]byte(txt), &res)
		if err != nil {
			break
		}
		w.Write([]byte("[model]: " + res.Candidates[0].Content.Parts[0].Text))
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "read stream failed.")
		return
	}
	fmt.Println("\nAI end." + str.String())
}
