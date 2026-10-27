package client

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"testing"

	mcpClient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// 这些是集成测试：需要一个启用了 MCP SSE 的服务端。
// 无服务端时必须 Skip 而不是 Fail，否则任何本地/CI 环境都跑不全 go test ./...
func newTestClient(t *testing.T) *mcpClient.Client {
	t.Helper()
	c, err := NewClient("http://localhost:8888/sse", "test-client", "1.0.0", "gin-vue-admin MCP服务")
	if err != nil {
		msg := err.Error()
		if strings.Contains(msg, "connect") || strings.Contains(msg, "transport") ||
			strings.Contains(msg, "refused") || strings.Contains(msg, "timeout") {
			t.Skipf("MCP SSE 服务不可达，跳过集成测试：%v", err)
		}
		t.Fatalf("创建 MCP 客户端失败: %v", err)
	}
	t.Cleanup(func() { _ = c.Close() })
	return c
}

// 测试 MCP 客户端连接
func TestMcpClientConnection(t *testing.T) {
	_ = newTestClient(t)
}

func TestTools(t *testing.T) {
	t.Run("currentTime", func(t *testing.T) {
		c := newTestClient(t)
		ctx := context.Background()

		request := mcp.CallToolRequest{}
		request.Params.Name = "currentTime"
		request.Params.Arguments = map[string]any{
			"timezone": "UTC+8",
		}

		result, err := c.CallTool(ctx, request)
		if err != nil {
			t.Fatalf("方法调用错误: %v", err)
		}

		if len(result.Content) != 1 {
			t.Errorf("应该有且仅返回1条信息，但是现在有 %d", len(result.Content))
		}
		if content, ok := result.Content[0].(mcp.TextContent); ok {
			t.Logf("成功返回信息%s", content.Text)
		} else {
			t.Logf("返回为止类型信息%+v", content)
		}
	})

	t.Run("getNickname", func(t *testing.T) {
		c := newTestClient(t)
		ctx := context.Background()

		request := mcp.CallToolRequest{}
		request.Params.Name = "getNickname"
		request.Params.Arguments = map[string]any{
			"username": "admin",
		}

		result, err := c.CallTool(ctx, request)
		if err != nil {
			t.Fatalf("方法调用错误: %v", err)
		}

		if len(result.Content) != 1 {
			t.Errorf("应该有且仅返回1条信息，但是现在有 %d", len(result.Content))
		}
		if content, ok := result.Content[0].(mcp.TextContent); ok {
			t.Logf("成功返回信息%s", content.Text)
		} else {
			t.Logf("返回为止类型信息%+v", content)
		}
	})
}

func TestGetTools(t *testing.T) {
	c := newTestClient(t)
	ctx := context.Background()

	toolsRequest := mcp.ListToolsRequest{}

	toolListResult, err := c.ListTools(ctx, toolsRequest)
	if err != nil {
		t.Fatalf("获取工具列表失败: %v", err)
	}
	for i := range toolListResult.Tools {
		tool := toolListResult.Tools[i]
		fmt.Printf("工具名称: %s\n", tool.Name)
		fmt.Printf("工具描述: %s\n", tool.Description)

		// 打印参数信息
		if tool.InputSchema.Properties != nil {
			fmt.Println("参数列表:")
			for paramName, prop := range tool.InputSchema.Properties {
				required := "否"
				// 检查参数是否在必填列表中
				if slices.Contains(tool.InputSchema.Required, paramName) {
					required = "是"
				}
				fmt.Printf("  - %s (类型: %s, 描述: %s, 必填: %s)\n",
					paramName, prop.(map[string]any)["type"], prop.(map[string]any)["description"], required)
			}
		} else {
			fmt.Println("该工具没有参数")
		}
		fmt.Println("-------------------")
	}
}
