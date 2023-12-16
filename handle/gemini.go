package handle

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var ctx = context.Background()

type content struct {
	Parts []string
	Role  string
}

func Gemini(w http.ResponseWriter, r *http.Request) {
	// Access your API key as an environment variable (see "Set up your API key" above)
	client, err := genai.NewClient(ctx, option.WithAPIKey(os.Getenv("GEMINI_API_KEY")))
	if err != nil {
		fmt.Fprintf(w, "genai new error.")
		return
	}
	defer client.Close()
	var params []content
	model := client.GenerativeModel("gemini-pro-vision")
	cs := model.StartChat()
	cs.History = []*genai.Content{}
	if err = json.NewDecoder(r.Body).Decode(&params); err != nil {
		fmt.Println(err)
		fmt.Fprint(w, "Invalid JSON format")
		return
	}
	for i := 0; i < len(params); i++ {
		parts := []genai.Part{}
		for j := 0; j < len(params[i].Parts); j++ {
			parts = append(parts, genai.Text(params[i].Parts[j]))
		}
		cs.History = append(cs.History, &genai.Content{
			Parts: parts,
			Role:  params[i].Role,
		})
	}
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
