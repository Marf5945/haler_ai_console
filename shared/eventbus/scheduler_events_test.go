package eventbus

import "testing"

// loop trace 不發工具卡；一般 trace 照發。
func TestIsTaskLoopTrace(t *testing.T) {
	for _, id := range []string{"chatroute-dag-1-n1", "clitask-dag-1-n2-r3"} {
		if !IsTaskLoopTrace(id) {
			t.Fatalf("%s should be loop trace", id)
		}
	}
	for _, id := range []string{"task-node-n1", "chat-123", "", "chatrout"} {
		if IsTaskLoopTrace(id) {
			t.Fatalf("%s should not be loop trace", id)
		}
	}
}
