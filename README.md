# eino-ext-llamaparse

[English](README.md) | [简体中文](README_zh.md)

An extension for [Eino](https://github.com/cloudwego/eino) that integrates [LlamaParse](https://cloud.llamaindex.ai/llamaparse) for high-quality document parsing.

## Installation

```bash
go get github.com/cloudwego/eino-ext-llamaparse
```

## Usage

```go
import (
    "context"
    "github.com/cloudwego/eino-ext-llamaparse/document/llamaparse"
)

func main() {
    ctx := context.Background()
    
    // Create a new LlamaParser
    parser := llamaparse.NewLlamaParser("your-llama-api-key")
    
    // Parse a document from a reader
    // LlamaParse relies on file extensions, so it's recommended to provide a filename
    docs, err := parser.Parse(ctx, reader, &llamaparse.ParseOptions{
        Filename: "example.pdf",
    })
    
    if err != nil {
        panic(err)
    }
    
    for _, doc := range docs {
        fmt.Println(doc.Content)
    }
}
```

### Options

You can customize the parser with `Option`:

```go
parser := llamaparse.NewLlamaParser(apiKey, 
    llamaparse.WithHTTPClient(customClient),
    llamaparse.WithCheckInterval(5 * time.Second),
    llamaparse.WithMaxTimeout(15 * time.Minute),
)
```

And customize each `Parse` call with `ParseOptions`:

```go
docs, err := parser.Parse(ctx, reader, &llamaparse.ParseOptions{
    Filename: "report.docx",
    Language: "en",
    ParsingInstruction: "Extract all tables in markdown format",
    ResultType: llamaparse.ResultTypeMarkdown,
})
```

## License

MIT
