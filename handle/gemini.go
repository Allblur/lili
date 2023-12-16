package handle

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var ctx = context.Background()

func Gemini(w http.ResponseWriter, r *http.Request) {
	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		fmt.Fprintf(w, "Params Don't null")
		return
	}
	defer client.Close()
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Fprintf(w, "Params Don't null")
		return
	}
	defer r.Body.Close()
	var params []*genai.Content
	err = json.Unmarshal(bytes, &params)
	if err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format")
		return
	}
	model := client.GenerativeModel("gemini-pro-vision")
	// Initialize the chat
	cs := model.StartChat()
	cs.History = params
	/* cs.History = []*genai.Content{
		&genai.Content{
			Parts: []genai.Part{
				genai.Text("Hello, I have 2 dogs in my house."),
			},
			Role: "user",
		},
		&genai.Content{
			Parts: []genai.Part{
				genai.Text("Great to meet you. What would you like to know?"),
			},
			Role: "model",
		},
	} */
	str := strings.Builder{}
	iter := cs.SendMessageStream(ctx, genai.Text("How many paws are in my house?"))
	for {
		resp, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			fmt.Fprintf(w, "load stream err.")
		}
		// fmt.Println(resp.Candidates[0].Content.Role, resp.Candidates[0].Content.Parts[0])
		content := fmt.Sprintf("%s: %+v", resp.Candidates[0].Content.Role, resp.Candidates[0].Content.Parts)
		str.WriteString(content)
		w.Write([]byte(content))
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		flusher.Flush()
	}
	fmt.Println("\nAI：" + str.String())
}
