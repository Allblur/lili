package handle

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

type VisionReqContents struct {
	Parts []any `json:"parts"`
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

var safetySettings = []SafetySettings{
	{
		Category:  "HARM_CATEGORY_DANGEROUS_CONTENT",
		Threshold: "BLOCK_ONLY_HIGH",
	},
}

func (h *Handle) Gapi(w http.ResponseWriter, r *http.Request) {
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
	url := fmt.Sprintf("%s%s/models/gemini-pro:streamGenerateContent?key=%s", h.GeminiApiUrl, reqBody.Version, h.GeminiApiKey)
	apiRequestBody := ApiRequestBody{
		ComonBody: ComonBody{
			Contents:         contents,
			GenerationConfig: reqBody.GenerationConfig,
		},
		SafetySettings: safetySettings,
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
	resp, err := h.HttpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "resp error.")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		fmt.Println(resp.Body)
		fmt.Fprint(w, "Invalid value.")
		return
	}
	str := strings.Builder{}
	// 处理stream结果
	scanner := makeScanner(resp.Body)
	for scanner.Scan() {
		txt := scanner.Text()
		txt = strings.TrimLeft(txt, "[,\r\n")
		txt = strings.TrimRight(txt, "],\r\n")
		str.WriteString(txt)
		var res GenerateContentResponse
		err = json.Unmarshal([]byte(txt), &res)
		if err != nil {
			break
		}
		w.Write([]byte(res.Candidates[0].Content.Parts[0].Text))
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

func (h *Handle) Gv(w http.ResponseWriter, r *http.Request) {
	reqBody := VisionReq{}
	if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format")
		return
	}
	url := fmt.Sprintf("%s%s/models/gemini-pro-vision:streamGenerateContent?key=%s", h.GeminiApiUrl, reqBody.Version, h.GeminiApiKey)
	// reqBody.Contents[0].Parts[0] = Parts{reqBody.Contents[0].Parts[0]}
	apiRequestBody := VisionApiReq{
		Contents: reqBody.Contents,
		GenerationConfig: GenerationConfig{
			StopSequences:   []string{"Title"},
			Temperature:     1.0,
			MaxOutputTokens: 4096,
			TopP:            0.8,
			TopK:            10,
		},
		SafetySettings: safetySettings,
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
	resp, err := h.HttpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "resp error.")
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		fmt.Println(resp.Body)
		fmt.Fprint(w, "Invalid value.")
		return
	}
	str := strings.Builder{}
	// 处理stream结果
	scanner := makeScanner(resp.Body)
	for scanner.Scan() {
		txt := scanner.Text()
		txt = strings.TrimLeft(txt, "[,\r\n")
		txt = strings.TrimRight(txt, "],\r\n")
		str.WriteString(txt)
		var res GenerateContentResponse
		err = json.Unmarshal([]byte(txt), &res)
		if err != nil {
			break
		}
		w.Write([]byte(res.Candidates[0].Content.Parts[0].Text))
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
	fmt.Println("\nGeminivision end." + str.String())
}

func makeScanner(r io.Reader) *bufio.Scanner {
	// 处理stream结果
	scanner := bufio.NewScanner(r)
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
	return scanner
}
