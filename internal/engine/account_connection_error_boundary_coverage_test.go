package engine

import (
	"context"
	"errors"
	"testing"

	"xianyu-go/internal/xianyu/ws"
)

// TestHandleWSConnectFailureClassifiesAllConnectionErrors 验证握手失败按服务端错误类型选择用户可见原因与重试行为。
func TestHandleWSConnectFailureClassifiesAllConnectionErrors(t *testing.T) {
	// cases 保存每种握手错误及其对应的运行状态原因与重试期望。
	cases := []struct {
		name      string
		err       error
		reason    string
		wantState string
		terminal  bool
	}{
		{name: "connect-limit", err: &ws.RegError{Kind: ws.RegErrorConnectLimit, Reason: "session remove"}, reason: "消息会话已被服务端移除", wantState: RuntimeAuthExpired, terminal: true},
		{name: "invalid-token", err: &ws.RegError{Kind: ws.RegErrorInvalidToken, Reason: "invalid token"}, reason: "消息凭证被拒绝，请重新登录", wantState: RuntimeAuthExpired, terminal: true},
		{name: "authentication", err: &ws.RegError{Kind: ws.RegErrorAuthentication, Reason: "not auth"}, reason: "消息凭证被拒绝，请重新登录", wantState: RuntimeAuthExpired, terminal: true},
		{name: "generic", err: errors.New("dial failed"), reason: "消息服务连接失败，正在重试", wantState: RuntimeReconnecting, terminal: false},
	}
	// testCase 表示当前子测试使用的一种握手错误分类样本。
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			// account 是仅用于验证错误分类的本地账号。
			account := New(Config{CookieID: testCase.name, CookieStr: "unb=1"})
			// returnedErr 是握手失败处理后的返回错误。
			returnedErr := account.handleWSConnectFailure(context.Background(), testCase.err)
			if testCase.terminal {
				if !errors.Is(returnedErr, testCase.err) {
					t.Fatalf("认证类错误应保留原始握手错误：got=%v want=%v", returnedErr, testCase.err)
				}
			} else {
				if returnedErr != nil {
					t.Fatalf("瞬时网络错误应返回 nil 允许重试：got=%v", returnedErr)
				}
			}
			// runtimeStatus 保存错误分类后供用户界面读取的账号状态。
			runtimeStatus := account.RuntimeStatus()
			if runtimeStatus.State != testCase.wantState || runtimeStatus.Message != testCase.reason {
				t.Fatalf("错误分类不正确：status=%+v wantState=%q wantReason=%q", runtimeStatus, testCase.wantState, testCase.reason)
			}
		})
	}
}
