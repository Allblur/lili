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

func (h *Handle) Completions(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get("api-key") == "" {
		fmt.Fprint(w, "Unauthorized")
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprint(w, "Invalid JSON Params.")
		return
	}

	payload := bytes.NewBuffer(body)

	req, err := http.NewRequest("POST", h.AzureApiUrl, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("api-key", r.Header.Get("api-key"))

	res, err := h.HttpClient.Do(req)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Network error, api")
		return
	}
	defer res.Body.Close()
	str := strings.Builder{}
	// 处理stream结果
	scanner := bufio.NewScanner(res.Body)

	headerData := "data: "
	headerLen := len(headerData)
	for scanner.Scan() {
		var chatCompletionStream ChatCompletionStreamResponse
		line := strings.TrimSpace(scanner.Text())
		// fmt.Println(line + "\n")
		if strings.HasPrefix(line, headerData) && line != "data: [DONE]" {
			// line = strings.TrimPrefix(line, headerData)
			err = json.Unmarshal([]byte(line[headerLen:]), &chatCompletionStream)
			if err == nil && chatCompletionStream.Choices != nil && chatCompletionStream.Choices[0].FinishReason != "stop" {
				content := chatCompletionStream.Choices[0].Delta.Content
				str.WriteString(content)
				w.Write([]byte(content))
				flusher, ok := w.(http.Flusher)
				if !ok {
					return
				}
				flusher.Flush()
				// w.(http.Flusher).Flush()
			}
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Read stream failed.")
		return
	}
	fmt.Println("\nAI：" + str.String())
}

func (h *Handle) GenerateImage(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{
		"code": "0",
		"msg":  "Invalid JSON Params.",
	}
	if r.Header.Get("api-key") == "" {
		data["msg"] = "Unauthorized"
		json.NewEncoder(w).Encode(data)
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		json.NewEncoder(w).Encode(data)
		return
	}

	payload := bytes.NewBuffer(body)

	req, err := http.NewRequest("POST", h.AzureApiGenerateImageUrl, payload)

	if err != nil {
		data["msg"] = "Network error. " + err.Error()
		json.NewEncoder(w).Encode(data)
		return
	}
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("api-key", r.Header.Get("api-key"))

	res, err := h.HttpClient.Do(req)
	if err != nil {
		data["msg"] = err.Error()
		json.NewEncoder(w).Encode(data)
		return
	}
	b, _ := io.ReadAll(res.Body)
	var result map[string]any
	json.Unmarshal(b, &result)
	result["code"] = "1"
	result["msg"] = "Success"
	json.NewEncoder(w).Encode(result)
}
