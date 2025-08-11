package utils

import (
	"bytes"
	"github.com/unidoc/unipdf/v3/extractor"
	"github.com/unidoc/unipdf/v3/model"
	"io"
	"strings"
)

// ExtractPDFText 使用UniPDF提取PDF文本
func ExtractPDFText(file io.Reader) (string, error) {
	// 创建内存缓冲区避免重复读取
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, file); err != nil {
		return "", err
	}

	// 创建PDF阅读器
	pdfReader, err := model.NewPdfReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		return "", err
	}

	// 提取文本
	var textBuilder strings.Builder
	if numPages, err := pdfReader.GetNumPages(); err == nil && numPages > 0 {
		for i := 1; i <= numPages; i++ {
			if page, err := pdfReader.GetPage(i); err == nil && page != nil {
				if ex, err := extractor.New(page); err == nil {
					if pageText, err := ex.ExtractText(); err == nil && len(pageText) > 0 {
						textBuilder.WriteString(strings.TrimSpace(pageText))
						textBuilder.WriteString("\n\n")
					}
				}
			}
		}
	}
	return textBuilder.String(), nil
}

// CombineMessages 简单拼接用户消息和PDF内容
func CombineMessages(userMsg, pdfContent string) string {
	const maxLength = 2047

	// 空内容直接返回用户消息
	if pdfContent == "" {
		return userMsg
	}

	// 检查PDF内容长度
	if len([]rune(pdfContent)) > maxLength {
		return userMsg + "\n[系统提示]pdf文本超出上下文2048限制"
	}

	return userMsg + "\n[PDF内容开始]" + pdfContent + "[PDF内容结束]"
}

// SplitText 将文本分割为指定大小的块
func SplitText(text string, maxChunkSize int) []string {
	var chunks []string
	if text == "" {
		return chunks
	}

	runes := []rune(text)
	totalLength := len(runes)

	for i := 0; i < totalLength; i += maxChunkSize {
		end := i + maxChunkSize
		if end > totalLength {
			end = totalLength
		}
		chunks = append(chunks, string(runes[i:end]))
	}

	return chunks
}

// TruncateText 截断文本到指定长度
func TruncateText(text string, maxLen int) string {
	runes := []rune(text)
	if len(runes) <= maxLen {
		return text
	}
	return string(runes[:maxLen]) + "..."
}
