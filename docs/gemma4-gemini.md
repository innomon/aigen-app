# How you can use Gemma 4 in Go.

Google’s modern Go SDK for the Gemini API is `github.com/google/generative-ai-go`. Make sure you have your API key set in your environment variables (`export GEMINI_API_KEY="your_key_here"`).

### 🛠️ Go Code Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	// 1. Retrieve the API key from your environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	// 2. Initialize the client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Error creating Gemini client: %v", err)
	}
	defer client.Close()

	// 3. Select the Gemma 4 model
	// Use "gemma-4-26b-a4b-it" for low latency or "gemma-4-31b-it" for raw quality
	model := client.GenerativeModel("gemma-4-26b-a4b-it")

	// 4. (Optional) Configure settings like temperature or Thinking Mode
	model.SetTemperature(0.7)
	// Note: Thinking Mode config can also be passed via the model's Config struct 
	// depending on the exact SDK minor version features enabled.

	// 5. Generate content
	prompt := "Explain the concept of Go routines and channels to a beginner in two sentences."
	fmt.Printf("Prompt: %s\n\nResponded:\n", prompt)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("Error generating content: %v", err)
	}

	// 6. Print the response
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Print(part)
			}
		}
	}
	fmt.Println()
}

```

**Thinking Mode Off:**

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	// 1. Retrieve the API key from your environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("GEMINI_API_KEY environment variable not set")
	}

	// 2. Initialize the client
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Error creating Gemini client: %v", err)
	}
	defer client.Close()

	// 3. Select the Gemma 4 model
	model := client.GenerativeModel("gemma-4-26b-a4b-it")

	// 4. Turn thinking mode OFF explicitly 
	// Setting the ThinkingBudget to 0 tells the API not to spend any tokens on reasoning.
	model.GenerationConfig = genai.GenerationConfig{
		Temperature: genai.Ptr(0.7),
		ThinkingConfig: &genai.ThinkingConfig{
			ThinkingBudget: genai.Ptr(0),
		},
	}

	// 5. Generate content
	prompt := "Explain the concept of Go routines and channels to a beginner in two sentences."
	fmt.Printf("Prompt: %s\n\nResponded (Thinking Mode: OFF):\n", prompt)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		log.Fatalf("Error generating content: %v", err)
	}

	// 6. Print the response
	for _, cand := range resp.Candidates {
		if cand.Content != nil {
			for _, part := range cand.Content.Parts {
				fmt.Print(part)
			}
		}
	}
	fmt.Println()
}

```

---

### 📦 Prerequisites & Running

1. **Initialize your module** and grab the correct dependencies:
```bash
go mod init gemma-demo
go get github.com/google/generative-ai-go/genai
go get google.golang.org/api/option

```


2. **Run the code**:
```bash
export GEMINI_API_KEY="your-actual-api-key"
go run main.go

```

## List Gemma Models

To find all available Gemma 4 variants on the Gemini API using Go, you can pull the model registry dynamically using the official `github.com/google/generative-ai-go` SDK.

Because the API returns *all* Google models (including Gemini, older Gemma variants, and embedding models), the code uses Go's `strings.Contains` to dynamically filter out any model string containing `"gemma-4"`.

### 🛠️ Go Implementation

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

func main() {
	ctx := context.Background()

	// 1. Get API Key from environment
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		log.Fatal("Please set your GEMINI_API_KEY environment variable.")
	}

	// 2. Initialize the client using the correct backend configuration
	client, err := genai.NewClient(ctx, option.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("Failed to initialize Gemini client: %v", err)
	}
	defer client.Close()

	fmt.Println("Fetching available Gemma 4 variants from the Gemini API...")
	fmt.Println("==========================================================")

	// 3. Iterate through all models available via the endpoint
	iter := client.ListModels(ctx)
	foundGemma4 := false

	for {
		modelInfo, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			log.Fatalf("Error while iterating models: %v", err)
		}

		// 4. Filter specifically for Gemma 4 models
		// The API paths look like "models/gemma-4-31b-it" or "models/gemma-4-26b-a4b-it"
		if strings.Contains(strings.ToLower(modelInfo.Name), "gemma-4") {
			foundGemma4 = true
			fmt.Printf("🔹 Model ID:     %s\n", modelInfo.Name)
			fmt.Printf("   Name:         %s\n", modelInfo.DisplayName)
			fmt.Printf("   Description:  %s\n", modelInfo.Description)
			fmt.Printf("   Input Limit:  %d tokens\n", modelInfo.InputTokenLimit)
			fmt.Printf("   Output Limit: %d tokens\n", modelInfo.OutputTokenLimit)
			fmt.Println("----------------------------------------------------------")
		}
	}

	if !foundGemma4 {
		fmt.Println("No Gemma 4 variants were found on this specific API account tier yet.")
	}
}

```

---

### 📦 Setup and Execution

1. Initialize your project directory and fetch the necessary modules:
```bash
go mod init gemma-list
go get github.com/google/generative-ai-go/genai
go get google.golang.org/api/iterator
go get google.golang.org/api/option

```


2. Make sure your API key is exported into your environment, then execute your script:
```bash
export GEMINI_API_KEY="AIzaSyYourKeyHere..."
go run main.go

```
