// Copyright (c) Microsoft. All rights reserved.

package agentframework

import (
	"context"
	"log/slog"
	"time"
)

// LoggingMiddleware returns an [AgentMiddleware] that logs agent runs using slog.
func LoggingMiddleware(logger *slog.Logger) AgentMiddleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next AgentHandler) AgentHandler {
		return func(ctx context.Context, req *AgentRequest) (*AgentResponse, error) {
			start := time.Now()
			logger.InfoContext(ctx, "agent run started",
				"message_count", len(req.Messages),
			)

			resp, err := next(ctx, req)

			duration := time.Since(start)
			if err != nil {
				logger.ErrorContext(ctx, "agent run failed",
					"duration", duration,
					"error", err,
				)
				return nil, err
			}

			logger.InfoContext(ctx, "agent run completed",
				"duration", duration,
				"response_messages", len(resp.Messages),
				"input_tokens", resp.Usage.InputTokens,
				"output_tokens", resp.Usage.OutputTokens,
			)
			return resp, nil
		}
	}
}
