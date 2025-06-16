package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type MCPServer struct {
	URL string `json:"url"`
}

type Config struct {
	MCPServers map[string]MCPServer `json:"mcpServers"`
}

func readConfig(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var c Config
	if err := json.NewDecoder(f).Decode(&c); err != nil {
		return nil, err
	}
	return &c, nil
}

func walkDir(root string, maxDepth int, prefix string, b *strings.Builder) error {
	if maxDepth < 0 {
		return nil
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return err
	}
	for i, entry := range entries {
		connector := "├──"
		if i == len(entries)-1 {
			connector = "└──"
		}
		line := fmt.Sprintf("%s%s %s", prefix, connector, entry.Name())
		if entry.IsDir() {
			line += "/"
		}
		b.WriteString(line + "\n")
		if entry.IsDir() {
			newPrefix := prefix
			if i == len(entries)-1 {
				newPrefix += "    "
			} else {
				newPrefix += "│   "
			}
			walkDir(filepath.Join(root, entry.Name()), maxDepth-1, newPrefix, b)
		}
	}
	return nil
}

func indexDir(path string, depth int) (string, error) {
	var b strings.Builder
	b.WriteString("Directory of " + path + ":\n")
	err := walkDir(path, depth, "", &b)
	return b.String(), err
}

func askQuestion(question string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", errors.New("OPENAI_API_KEY not set")
	}
	client := openai.NewClient(apiKey)
	resp, err := client.CreateChatCompletion(context.Background(), openai.ChatCompletionRequest{
		Model:    "gpt-3.5-turbo",
		Messages: []openai.ChatCompletionMessage{{Role: "user", Content: question}},
	})
	if err != nil {
		return "", err
	}
	if len(resp.Choices) == 0 {
		return "", errors.New("no response")
	}
	return resp.Choices[0].Message.Content, nil
}

func callMCP(server, tool, paramsJSON, configPath string) (string, error) {
	cfg, err := readConfig(configPath)
	if err != nil {
		return "", err
	}
	srv, ok := cfg.MCPServers[server]
	if !ok {
		return "", fmt.Errorf("server %s not found", server)
	}
	url := fmt.Sprintf("%s/tool/%s", srv.URL, tool)
	req, err := http.NewRequest("POST", url, strings.NewReader(paramsJSON))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("error %s", string(body))
	}
	return string(body), nil
}

func main() {
	action := flag.String("action", "index", "index|ask|mcp")
	dir := flag.String("dir", ".", "directory to index")
	depth := flag.Int("depth", 2, "max depth")
	question := flag.String("question", "", "question for LLM")
	server := flag.String("server", "", "mcp server name")
	tool := flag.String("tool", "", "mcp tool name")
	params := flag.String("params", "{}", "tool params as JSON")
	config := flag.String("config", os.ExpandEnv("$HOME/.void/mcp.json"), "path to mcp config")
	flag.Parse()

	switch *action {
	case "index":
		res, err := indexDir(*dir, *depth)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		fmt.Print(res)
	case "ask":
		if *question == "" {
			fmt.Println("question required")
			os.Exit(1)
		}
		ans, err := askQuestion(*question)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		fmt.Println(ans)
	case "mcp":
		if *server == "" || *tool == "" {
			fmt.Println("server and tool required")
			os.Exit(1)
		}
		res, err := callMCP(*server, *tool, *params, *config)
		if err != nil {
			fmt.Println("error:", err)
			os.Exit(1)
		}
		fmt.Println(res)
	default:
		fmt.Println("unknown action")
		os.Exit(1)
	}
}
