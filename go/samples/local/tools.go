// Copyright (c) Microsoft. All rights reserved.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	af "github.com/microsoft/agent-framework/go/agentframework"

	"golang.org/x/sys/windows"
)

// GetTools returns the tool definitions for the local assistant.
func GetTools() []af.Tool {
	weatherTool := af.NewTypedTool("get_weather",
		"Get the current weather for a location.",
		func(ctx context.Context, args struct {
			Location string `json:"location" jsonschema:"description=City name or location,required"`
			Unit     string `json:"unit"     jsonschema:"description=Temperature unit,enum=celsius|fahrenheit"`
		}) (any, error) {
			unit := args.Unit
			if unit == "" {
				unit = "fahrenheit"
			}
			temp := 72
			if unit == "celsius" {
				temp = 22
			}
			return map[string]any{
				"location":    args.Location,
				"temperature": temp,
				"unit":        unit,
				"condition":   "sunny",
			}, nil
		},
	)

	timeTool := af.NewTool("get_time",
		"Get the current time.",
		json.RawMessage(`{"type":"object","properties":{}}`),
		func(ctx context.Context, args json.RawMessage) (any, error) {
			now := time.Now()
			return map[string]string{
				"time":     now.Format("3:04 PM"),
				"date":     now.Format("Monday, January 2, 2006"),
				"timezone": now.Location().String(),
				"iso8601":  now.Format(time.RFC3339),
			}, nil
		},
	)

	return []af.Tool{weatherTool, timeTool, listFilesTool(), listDockerImagesTool(), diskSpaceTool()}
}

// listFilesTool returns the list_local_files tool that lists files in the
// current working directory. It accepts no arguments and returns only
// filenames (no paths, no traversal).
func listFilesTool() af.Tool {
	return af.NewTool(
		"list_local_files",
		"Lists files in the current working directory of the agent runtime",
		json.RawMessage(`{"type":"object","properties":{},"required":[]}`),
		func(ctx context.Context, args json.RawMessage) (any, error) {
			// Reject non-empty arguments.
			if len(args) > 0 {
				var m map[string]any
				if err := json.Unmarshal(args, &m); err == nil && len(m) > 0 {
					return nil, &af.ToolError{
						ToolName: "list_local_files",
						Message:  "this tool does not accept arguments",
					}
				}
			}

			wd, err := os.Getwd()
			if err != nil {
				return nil, &af.ToolError{
					ToolName: "list_local_files",
					Message:  "failed to get working directory",
				}
			}

			entries, err := os.ReadDir(wd)
			if err != nil {
				return nil, &af.ToolError{
					ToolName: "list_local_files",
					Message:  "failed to read directory",
				}
			}

			files := make([]string, 0, len(entries))
			for _, e := range entries {
				files = append(files, e.Name())
			}

			return map[string]any{"files": files}, nil
		},
	)
}

// listDockerImagesTool returns a tool that lists Docker images on the host.
func listDockerImagesTool() af.Tool {
	return af.NewTool(
		"list_docker_images",
		"Lists Docker images available on the host machine",
		json.RawMessage(`{"type":"object","properties":{}}`),
		func(ctx context.Context, args json.RawMessage) (any, error) {
			cmd := exec.CommandContext(ctx, "docker", "images", "--format", "{{.Repository}}\t{{.Tag}}\t{{.ID}}\t{{.Size}}\t{{.CreatedSince}}")
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			if err := cmd.Run(); err != nil {
				return nil, &af.ToolError{
					ToolName: "list_docker_images",
					Message:  fmt.Sprintf("docker command failed: %v — %s", err, stderr.String()),
				}
			}

			var images []map[string]string
			for _, line := range strings.Split(strings.TrimSpace(stdout.String()), "\n") {
				if line == "" {
					continue
				}
				parts := strings.SplitN(line, "\t", 5)
				img := map[string]string{"repository": "", "tag": "", "id": "", "size": "", "created": ""}
				if len(parts) > 0 { img["repository"] = parts[0] }
				if len(parts) > 1 { img["tag"] = parts[1] }
				if len(parts) > 2 { img["id"] = parts[2] }
				if len(parts) > 3 { img["size"] = parts[3] }
				if len(parts) > 4 { img["created"] = parts[4] }
				images = append(images, img)
			}

			return map[string]any{
				"count":  len(images),
				"images": images,
			}, nil
		},
	)
}

// diskSpaceTool returns a tool that reports disk space for all drives.
func diskSpaceTool() af.Tool {
	return af.NewTool(
		"get_disk_space",
		"Gets available disk space for all drives on the host machine",
		json.RawMessage(`{"type":"object","properties":{}}`),
		func(ctx context.Context, args json.RawMessage) (any, error) {
			var drives []map[string]any

			for _, letter := range "ABCDEFGHIJKLMNOPQRSTUVWXYZ" {
				root := string(letter) + ":\\"
				rootPtr, _ := windows.UTF16PtrFromString(root)

				var freeBytesAvailable, totalBytes, totalFreeBytes uint64
				err := windows.GetDiskFreeSpaceEx(rootPtr, &freeBytesAvailable, &totalBytes, &totalFreeBytes)
				if err != nil {
					continue // drive doesn't exist or isn't ready
				}

				totalKB := totalBytes / 1024
				freeKB := totalFreeBytes / 1024
				usedKB := totalKB - freeKB

				drives = append(drives, map[string]any{
					"drive":          root,
					"total_readable":  formatBytes(totalBytes),
					"free_readable":   formatBytes(totalFreeBytes),
					"used_readable":   formatBytes(totalBytes - totalFreeBytes),
					"total_kb":        totalKB,
					"free_kb":         freeKB,
					"used_kb":         usedKB,
					"percent_used":    fmt.Sprintf("%.1f%%", float64(usedKB)/float64(totalKB)*100),
				})
			}

			if len(drives) == 0 {
				return nil, &af.ToolError{
					ToolName: "get_disk_space",
					Message:  "no drives found",
				}
			}

			return map[string]any{"drives": drives}, nil
		},
	)
}

func formatBytes(b uint64) string {
	switch {
	case b >= 1<<40:
		return fmt.Sprintf("%.1f TB", float64(b)/float64(1<<40))
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

// ToolCallLoggingMiddleware returns a FunctionMiddleware that logs tool
// invocations with a consistent [agent] prefix.
func ToolCallLoggingMiddleware() af.FunctionMiddleware {
	return func(next af.FunctionHandler) af.FunctionHandler {
		return func(ctx context.Context, tool af.Tool, args json.RawMessage) (any, error) {
			log.Printf("[agent] tool call requested: %s", tool.Name())

			result, err := next(ctx, tool, args)
			if err != nil {
				log.Printf("[agent] tool error: %s: %v", tool.Name(), err)
				return result, err
			}

			// Log file count for list_local_files.
			if m, ok := result.(map[string]any); ok {
				if f, ok := m["files"].([]string); ok {
					log.Printf("[agent] tool executed successfully (%d files)", len(f))
					return result, nil
				}
			}
			log.Printf("[agent] tool executed successfully")
			return result, nil
		}
	}
}
