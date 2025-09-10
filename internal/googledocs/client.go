package googledocs

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

type Client struct {
	service *docs.Service
}

func NewClient(ctx context.Context, credentialsFile string) (*Client, error) {
	service, err := docs.NewService(ctx, option.WithCredentialsFile(credentialsFile))
	if err != nil {
		return nil, fmt.Errorf("failed to create Google Docs service: %w", err)
	}

	return &Client{
		service: service,
	}, nil
}

func (c *Client) GetDocumentContent(ctx context.Context, docID string) (string, error) {
	doc, err := c.service.Documents.Get(docID).Context(ctx).Do()
	if err != nil {
		return "", fmt.Errorf("failed to get document: %w", err)
	}

	return extractTextFromDocument(doc), nil
}

func extractTextFromDocument(doc *docs.Document) string {
	var text strings.Builder

	for _, element := range doc.Body.Content {
		extractTextFromStructuralElement(element, &text)
	}

	return strings.TrimSpace(text.String())
}

func extractTextFromStructuralElement(element *docs.StructuralElement, text *strings.Builder) {
	if element.Paragraph != nil {
		extractTextFromParagraph(element.Paragraph, text)
	}
	if element.Table != nil {
		extractTextFromTable(element.Table, text)
	}
	if element.SectionBreak != nil {
		text.WriteString("\n")
	}
}

func extractTextFromParagraph(paragraph *docs.Paragraph, text *strings.Builder) {
	for _, element := range paragraph.Elements {
		if element.TextRun != nil {
			text.WriteString(element.TextRun.Content)
		}
	}
}

func extractTextFromTable(table *docs.Table, text *strings.Builder) {
	for _, row := range table.TableRows {
		for _, cell := range row.TableCells {
			for _, element := range cell.Content {
				extractTextFromStructuralElement(element, text)
			}
			text.WriteString("\t")
		}
		text.WriteString("\n")
	}
}
