//go:build darwin

package visual_learning

// 這個檔只放 cgo //export 回呼。依 cgo 規則，含 //export 的檔其 C preamble
// 只能有「宣告」不能有「定義」，故所有 C 函式定義都放在 native_input_darwin.go，
// 這裡的 preamble 僅 include 標準標頭（純宣告）。

// #include <stdint.h>
import "C"

// goNativeClickCallback 由 native_input_darwin.go 的 CGEventTap C 回呼呼叫，
// 透過 cgo.Handle 還原 *NativeInput 後送出事件。clickCount 來自
// kCGMouseEventClickState（雙擊時第二次 mouse-up 為 2）。
//
//export goNativeClickCallback
func goNativeClickCallback(x, y C.double, button, clickCount C.int, handle C.uintptr_t) {
	n, ok := handleToNativeInput(uintptr(handle))
	if !ok || n == nil {
		return
	}
	n.emitClick(int(x), int(y), int(button), int(clickCount))
}
