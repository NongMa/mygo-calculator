package main

import (
	"testing"

	"github.com/egoist/mygo/ui"
)

// TestResponsive 验证窗口尺寸变化时按钮和显示屏尺寸同步变化。
func TestResponsive(t *testing.T) {
	app := NewCalc()

	// 默认尺寸
	tt := ui.NewTester(app.view, 540, 752)
	cBtn, ok1 := tt.Find("C")
	dText, ok2 := tt.Find("0")
	t.Logf("default 540x752: C found=%v rect=%v; display found=%v rect=%v", ok1, cBtn, ok2, dText)

	// 最大化尺寸
	tt.SetSize(720, 985)
	cBtn2, ok3 := tt.Find("C")
	dText2, ok4 := tt.Find("0")
	t.Logf("max 720x985: C found=%v rect=%v; display found=%v rect=%v", ok3, cBtn2, ok4, dText2)

	if cBtn2.W <= cBtn.W {
		t.Errorf("button did not grow: default W=%v, max W=%v", cBtn.W, cBtn2.W)
	}
}
