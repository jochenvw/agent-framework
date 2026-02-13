// Copyright (c) Microsoft. All rights reserved.

package agentframework

// ContentType identifies the kind of content within a message.
type ContentType string

const (
	ContentTypeText                   ContentType = "text"
	ContentTypeTextReasoning          ContentType = "reasoning"
	ContentTypeData                   ContentType = "data"
	ContentTypeURI                    ContentType = "uri"
	ContentTypeError                  ContentType = "error"
	ContentTypeFunctionCall           ContentType = "functionCall"
	ContentTypeFunctionResult         ContentType = "functionResult"
	ContentTypeUsage                  ContentType = "usage"
	ContentTypeHostedFile             ContentType = "hostedFile"
	ContentTypeHostedVectorStore      ContentType = "hostedVectorStore"
	ContentTypeCodeInterpreterCall    ContentType = "codeInterpreterToolCall"
	ContentTypeCodeInterpreterResult  ContentType = "codeInterpreterToolResult"
	ContentTypeImageGenCall           ContentType = "imageGenerationToolCall"
	ContentTypeImageGenResult         ContentType = "imageGenerationToolResult"
	ContentTypeMCPServerCall          ContentType = "mcpServerToolCall"
	ContentTypeMCPServerResult        ContentType = "mcpServerToolResult"
	ContentTypeApprovalRequest        ContentType = "functionApprovalRequest"
	ContentTypeApprovalResponse       ContentType = "functionApprovalResponse"
)

// Content is a sealed interface representing a piece of content within a [Message].
// Each concrete type carries data specific to its [ContentType].
// Use a type switch to inspect the underlying type.
type Content interface {
	// Type returns the discriminator for this content item.
	Type() ContentType

	// sealed prevents external implementations.
	sealed()
}

// base is embedded by every concrete Content type to satisfy the sealed marker.
type base struct{}

func (base) sealed() {}

// TextContent holds plain text.
type TextContent struct {
	base
	Text string
}

func (c *TextContent) Type() ContentType { return ContentTypeText }

// TextReasoningContent holds chain-of-thought / reasoning text.
type TextReasoningContent struct {
	base
	Text string
}

func (c *TextReasoningContent) Type() ContentType { return ContentTypeTextReasoning }

// DataContent holds binary data represented as a data URI.
type DataContent struct {
	base
	URI       string // data URI (e.g. data:image/png;base64,...)
	MediaType string
}

func (c *DataContent) Type() ContentType { return ContentTypeData }

// URIContent holds an external URI reference.
type URIContent struct {
	base
	URI       string
	MediaType string
}

func (c *URIContent) Type() ContentType { return ContentTypeURI }

// ErrorContent represents an error returned as message content.
type ErrorContent struct {
	base
	Message   string
	ErrorCode string
	Details   any
}

func (c *ErrorContent) Type() ContentType { return ContentTypeError }

// FunctionCallContent represents a tool/function call requested by the model.
type FunctionCallContent struct {
	base
	CallID    string
	Name      string
	Arguments string // JSON-encoded arguments
}

func (c *FunctionCallContent) Type() ContentType { return ContentTypeFunctionCall }

// FunctionResultContent represents the result of a tool/function call.
type FunctionResultContent struct {
	base
	CallID string
	Result any
}

func (c *FunctionResultContent) Type() ContentType { return ContentTypeFunctionResult }

// UsageContent carries token usage information.
type UsageContent struct {
	base
	Usage UsageDetails
}

func (c *UsageContent) Type() ContentType { return ContentTypeUsage }

// HostedFileContent references a service-hosted file.
type HostedFileContent struct {
	base
	FileID string
}

func (c *HostedFileContent) Type() ContentType { return ContentTypeHostedFile }

// HostedVectorStoreContent references a service-hosted vector store.
type HostedVectorStoreContent struct {
	base
	VectorStoreID string
}

func (c *HostedVectorStoreContent) Type() ContentType { return ContentTypeHostedVectorStore }

// CodeInterpreterCallContent represents a code interpreter tool invocation.
type CodeInterpreterCallContent struct {
	base
	CallID string
	Code   string
}

func (c *CodeInterpreterCallContent) Type() ContentType { return ContentTypeCodeInterpreterCall }

// CodeInterpreterResultContent represents the output of a code interpreter.
type CodeInterpreterResultContent struct {
	base
	CallID string
	Output string
}

func (c *CodeInterpreterResultContent) Type() ContentType { return ContentTypeCodeInterpreterResult }

// ImageGenCallContent represents an image generation tool invocation.
type ImageGenCallContent struct {
	base
	CallID string
	Prompt string
}

func (c *ImageGenCallContent) Type() ContentType { return ContentTypeImageGenCall }

// ImageGenResultContent represents the output of an image generation tool.
type ImageGenResultContent struct {
	base
	CallID string
	URI    string
}

func (c *ImageGenResultContent) Type() ContentType { return ContentTypeImageGenResult }

// MCPServerCallContent represents an MCP server tool invocation.
type MCPServerCallContent struct {
	base
	CallID    string
	Name      string
	Arguments string
}

func (c *MCPServerCallContent) Type() ContentType { return ContentTypeMCPServerCall }

// MCPServerResultContent represents the output of an MCP server tool.
type MCPServerResultContent struct {
	base
	CallID string
	Result any
}

func (c *MCPServerResultContent) Type() ContentType { return ContentTypeMCPServerResult }

// ApprovalRequestContent requests user approval before invoking a tool.
type ApprovalRequestContent struct {
	base
	CallID    string
	Name      string
	Arguments string
}

func (c *ApprovalRequestContent) Type() ContentType { return ContentTypeApprovalRequest }

// ApprovalResponseContent carries the user's approval decision.
type ApprovalResponseContent struct {
	base
	CallID   string
	Approved bool
	Reason   string
}

func (c *ApprovalResponseContent) Type() ContentType { return ContentTypeApprovalResponse }
