# eino-ext-llamaparse

[English](README.md) | [简体中文](README_zh.md)

[Eino](https://github.com/cloudwego/eino) 的 [LlamaParse](https://cloud.llamaindex.ai/llamaparse) 扩展库。利用 LlamaParse 强大的多模态解析能力，将复杂的 PDF、图片、Word 等文档高质量地转换为 Markdown、文本或 JSON 格式，并无缝对接 Eino 的 Document Parser 接口。

## 特性

- **高质量解析**: 借助 LlamaParse，支持复杂表格、图片和多栏布局的精准解析。
- **Eino 集成**: 完美适配 `document.Parser` 接口，可直接在 Eino 的 Loader/Transformer 流程中使用。
- **自动轮询**: 内置解析任务的状态轮询机制。
- **高度可定制**: 支持自定义 HTTP 客户端、轮询间隔、解析指令等。

## 安装

```bash
go get github.com/alan22333/eino-ext-llamaparse
```

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"os"
	"github.com/alan22333/eino-ext-llamaparse/document/llamaparse"
)

func main() {
	ctx := context.Background()
	
	// 初始化 Parser
	parser := llamaparse.NewLlamaParser("你的-Llama-API-Key")
	
	// 从文件读取
	file, err := os.Open("example.pdf")
	if err != nil {
		panic(err)
	}
	defer file.Close()
	
	// 执行解析
	// 建议在 ParseOptions 中提供文件名，以帮助 LlamaParse 识别格式
	docs, err := parser.Parse(ctx, file, &llamaparse.ParseOptions{
		Filename: "example.pdf",
	})
	if err != nil {
		panic(err)
	}
	
	// 处理解析结果
	for _, doc := range docs {
		fmt.Println(doc.Content)
	}
}
```

## 进阶配置

### 全局配置 (NewLlamaParser)

```go
parser := llamaparse.NewLlamaParser(apiKey, 
    llamaparse.WithHTTPClient(customClient),    // 自定义 HTTP Client (如设置代理)
    llamaparse.WithCheckInterval(5 * time.Second), // 自定义轮询间隔
    llamaparse.WithMaxTimeout(15 * time.Minute),   // 最大解析超时时间
)
```

### 单次解析配置 (ParseOptions)

```go
docs, err := parser.Parse(ctx, reader, &llamaparse.ParseOptions{
    Filename: "report.docx",
    Language: "zh",                             // 指定文档语言
    ParsingInstruction: "将所有表格提取为 Markdown 格式", // 自定义解析指令
    ResultType: llamaparse.ResultTypeMarkdown,  // 输出格式: markdown, text, json
})
```

## 许可证

MIT
