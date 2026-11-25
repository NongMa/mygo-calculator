package main

import (
	_ "embed"
	"log"
	"math"
	"strconv"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

//go:embed icons/history.svg
var historySVG []byte

var historyIcon = ui.MustParseSVG(historySVG)

// Calc 是计算器的状态机，每帧由 view 从它读取状态并渲染界面。
type Calc struct {
	display     string   // 屏幕显示
	operand     float64
	operator    string   // + - * /
	waiting     bool     // 是否正在输入第二个操作数
	hasDot      bool     // 当前数是否已含小数点
	lastExpr    string   // 按 = 后的完整表达式
	records     []string // 计算记录
	showRecords bool     // 是否显示记录弹窗
}

func NewCalc() *Calc {
	return &Calc{display: "0"}
}

// onPress 处理按键，修改 Calc 状态。下一帧 view 会读取新状态重绘。
func (c *Calc) onPress(key string) {
	// 界面显示 × ÷，内部运算仍用 * /
	switch key {
	case "×":
		key = "*"
	case "÷":
		key = "/"
	}
	switch {
	case isDigit(key):
		c.pressDigit(key)
	case key == ".":
		c.pressDot()
	case key == "C":
		c.pressClear()
	case key == "±":
		c.pressSign()
	case key == "%":
		c.pressPercent()
	case key == "=":
		c.pressEqual()
	case isOperator(key):
		c.pressOperator(key)
	}
}

func isDigit(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}

func isOperator(s string) bool {
	return s == "+" || s == "-" || s == "*" || s == "/"
}

func (c *Calc) pressDigit(d string) {
	if c.waiting {
		c.display = d
		c.waiting = false
		c.hasDot = false
	} else {
		if c.display == "0" {
			c.display = d
		} else {
			c.display += d
		}
	}
}

func (c *Calc) pressDot() {
	if c.waiting {
		c.display = "0."
		c.waiting = false
		c.hasDot = true
		return
	}
	if !c.hasDot {
		c.display += "."
		c.hasDot = true
	}
}

func (c *Calc) pressClear() {
	c.display = "0"
	c.operand = 0
	c.operator = ""
	c.waiting = false
	c.hasDot = false
	c.lastExpr = "" // 归零时清除表达式
}

func (c *Calc) pressSign() {
	if val, err := strconv.ParseFloat(c.display, 64); err == nil {
		c.display = strconv.FormatFloat(-val, 'f', -1, 64)
	}
}

func (c *Calc) pressPercent() {
	if val, err := strconv.ParseFloat(c.display, 64); err == nil {
		c.display = strconv.FormatFloat(val/100, 'f', -1, 64)
	}
}

func (c *Calc) pressOperator(op string) {
	val, _ := strconv.ParseFloat(c.display, 64)
	if c.operator != "" && !c.waiting {
		// 连续按运算符：先算出中间结果
		c.operand = c.calculate(c.operand, val, c.operator)
		c.display = formatResult(c.operand)
	} else {
		c.operand = val
	}
	c.operator = op
	c.waiting = true
	c.hasDot = false
}

func (c *Calc) pressEqual() {
	if c.operator == "" {
		return
	}
	val, _ := strconv.ParseFloat(c.display, 64)
	expr := formatResult(c.operand) + " " + displayOp(c.operator) + " " + formatResult(val)
	result := c.calculate(c.operand, val, c.operator)
	resultStr := formatResult(result)
	c.display = resultStr
	c.lastExpr = expr + " ="
	c.records = append(c.records, expr+" = "+resultStr)
	if len(c.records) > 20 {
		c.records = c.records[len(c.records)-20:]
	}
	c.operand = result
	c.operator = ""
	c.waiting = true
	c.hasDot = strings.Contains(c.display, ".")
}

func (c *Calc) calculate(a, b float64, op string) float64 {
	switch op {
	case "+":
		return a + b
	case "-":
		return a - b
	case "*":
		return a * b
	case "/":
		if b == 0 {
			return 0
		}
		return a / b
	}
	return b
}

// formatResult 格式化结果：绝对值 >= 1e10 或 (非 0 且 < 1e-6) 时用科学计数法，
// 避免超长数字撑爆显示屏。
func formatResult(v float64) string {
	if v == 0 {
		return "0"
	}
	abs := math.Abs(v)
	if abs >= 1e10 || abs < 1e-6 {
		s := strconv.FormatFloat(v, 'e', 6, 64) // 如 6.999999e+24
		// 去掉尾随的零，让显示更简洁
		if i := strings.Index(s, "e"); i > 0 {
			mantissa := strings.TrimRight(strings.TrimRight(s[:i], "0"), ".")
			return mantissa + s[i:]
		}
		return s
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// displayOp 把内部运算符转成界面符号
func displayOp(op string) string {
	switch op {
	case "*":
		return "×"
	case "/":
		return "÷"
	default:
		return op
	}
}

// textWidth 用 ui.Shape 测量文本在给定字号下的实际像素宽度（DIP）。
func textWidth(s string, size float32) float32 {
	font := ui.Font{Family: "monospace", Size: size}
	glyphs := ui.Shape(s, font)
	if len(glyphs) == 0 {
		return 0
	}
	// 最后一个字形的 X 加上它的前进宽度即为整行宽度
	last := glyphs[len(glyphs)-1]
	return last.X + last.Advance
}

// fitFontSize 仅当文本实际宽度超过 maxWidth 时才缩小字号，避免明明放得下却被缩小。
func fitFontSize(s string, base, min, maxWidth float32) float32 {
	if s == "" {
		return base
	}
	w := textWidth(s, base)
	if w <= maxWidth {
		return base // 放得下，不缩小
	}
	f := base * maxWidth / w
	if f < min {
		return min
	}
	return f
}

// clamp 把 v 限制在 [lo, hi] 区间，用于响应式尺寸。
func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// clampInt 是 int 版本的 clamp，用于窗口尺寸边界。
func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// view 每帧被 MyGo 调用，根据当前状态构建界面。
func (c *Calc) view(ctx *ui.Context) {
	white := ui.Hex("#ffffff")
	dark := ui.Hex("#1c1c1e")
	muted := ui.RGBA(255, 255, 255, 140)

	// 响应式：显示屏按窗口高度比例占固定高度，按钮在剩余空间内自适应放大，
	// 两者都随窗口尺寸变化，且受 max 边界约束。
	width, height := ctx.Size()
	pad := clamp(width*0.06, 12, 24)
	gap := clamp(width*0.025, 6, 12)
	displayH := clamp(height*0.3, 150, 320)

	// 按宽度：4 个按钮 + 3 个间距填满内容宽度
	btnByWidth := (width - 2*pad - 3*gap) / 4
	// 按高度：窗口高 - 显示屏 - 边距 - 间距 - 底部文字，5 行按钮平分
	btnByHeight := (height - displayH - 2*pad - 6*gap - 24) / 5
	btnSize := clamp(min(btnByWidth, btnByHeight), 36, 180)

	zeroW := btnSize*2 + gap
	btnFont := clamp(btnSize*0.38, 16, 44)
	dispFont := clamp(btnSize*0.72, 30, 76)
	exprFont := clamp(btnFont*0.7, 12, 26)

	// 外层 Box：先绘制底层彩色背景，再绘制上层玻璃
	ui.Box(ctx).Fill().Children(func() {
		// 底层：Absolute 铺满 + 沉底，不占文档流，让内容从窗口顶部开始
		ui.Box(ctx).Absolute().Top(0).Left(0).Right(0).Bottom(0).ZIndex(-1).
			Gradient(ui.Hex("#1a1a3e"), ui.Hex("#2d1b69"), 135).Children(func() {
			ui.Box(ctx).Absolute().Top(-40).Left(-40).Size(200, 200).Radius(100).
				RadialGradient(ui.RadialGradient{
					Stops: []ui.GradientStop{{ui.Hex("#ff9f0a"), 0}, {ui.Transparent, 1}},
				})
			ui.Box(ctx).Absolute().Bottom(-60).Right(-40).Size(220, 220).Radius(110).
				RadialGradient(ui.RadialGradient{
					Stops: []ui.GradientStop{{ui.Hex("#0a84ff"), 0}, {ui.Transparent, 1}},
				})
			ui.Box(ctx).Absolute().Top(240).Right(-20).Size(160, 160).Radius(80).
				RadialGradient(ui.RadialGradient{
					Stops: []ui.GradientStop{{ui.Hex("#ff375f"), 0}, {ui.Transparent, 1}},
				})
		})

		// 上层：Liquid Glass 计算器，宽度跟随窗口并居中，最大化时按钮和显示屏同步放大
		ui.Column(ctx).Fill().Padding(pad).Gap(gap).Children(func() {
			// 显示屏：按窗口高度比例的固定响应式高度，始终有空间，不被挤压
			ui.Column(ctx).FillWidth().Height(displayH).Padding(12).Radius(16).
				Material(glass.Glass{}).Children(func() {
				// 右上角记录图标，与表达式分行避免重叠
				ui.Row(ctx).FillWidth().Justify(ui.End).Children(func() {
					if ui.Button(ctx, "").Label("计算记录").Tooltip("查看计算记录").Padding(6).Radius(8).
						Background(ui.Transparent).Border(0, ui.Transparent).NoShadow().
						Children(func() {
							ui.Icon(ctx, historyIcon).FontSize(20).TextColor(white)
						}).Clicked() {
						c.showRecords = true
					}
				})
				expr := c.lastExpr
				if c.operator != "" {
					if c.waiting {
						expr = formatResult(c.operand) + " " + displayOp(c.operator)
					} else {
						expr = formatResult(c.operand) + " " + displayOp(c.operator) + " " + c.display
					}
				}
				// 显示屏可用宽度 = 内容宽 - 面板两侧内边距(12*2) - 图标行占位余量
				displayAvail := width - 2*pad - 24 - 8
				ui.Row(ctx).FillWidth().Justify(ui.End).Children(func() {
					ui.Text(ctx, expr).FontSize(fitFontSize(expr, exprFont, 10, displayAvail)).TextColor(muted)
				})
				ui.Row(ctx).FillWidth().Justify(ui.End).Children(func() {
					ui.Text(ctx, c.display).FontSize(fitFontSize(c.display, dispFont, 16, displayAvail)).Bold().TextColor(white)
				})
			})

			rows := [][]string{
				{"C", "±", "%", "÷"},
				{"7", "8", "9", "×"},
				{"4", "5", "6", "-"},
				{"1", "2", "3", "+"},
				{"0", ".", "="},
			}

			for _, row := range rows {
				ui.Row(ctx).FillWidth().Gap(gap).Justify(ui.Center).Children(func() {
					for _, label := range row {
						w := btnSize
						if label == "0" {
							w = zeroW // “0”键占两列
						}
						var g glass.Glass
						var fg ui.Color
						switch {
						case isDigit(label) || label == ".":
							g = glass.Glass{Interactive: true}
							fg = white
						case label == "C" || label == "±" || label == "%":
							g = glass.Glass{Tint: ui.Hex("#a5a5aa"), Interactive: true}
							fg = dark
						default: // + - × ÷ =
							g = glass.Glass{Tint: ui.Hex("#ff9f0a"), Interactive: true}
							fg = white
						}
						btn := ui.Button(ctx, label).
							Width(w).Height(btnSize).
							Radius(999).
							Material(g).
							FontSize(btnFont).
							TextColor(fg)
						if btn.Clicked() {
							c.onPress(label)
						}
					}
				})
			}

			ui.Row(ctx).FillWidth().Justify(ui.Center).Children(func() {
				ui.Text(ctx, "MyGo Calculator").FontSize(11).TextColor(muted)
			})
		})

		// 完整记录弹窗：放在最外层，backdrop 蒙层覆盖整个窗口而非受 Column 内边距限制
		ui.DialogBase(ctx, &c.showRecords, func(backdrop, panel ui.Element) {
			backdrop.Background(ui.RGBA(0, 0, 0, 0.5))
			panel.Width(clamp(width-48, 240, 320)).Padding(16).Radius(20).
				Material(glass.Glass{Clarity: 0.4})
			ui.Text(ctx, "计算记录").FontSize(18).Bold().TextColor(white)
			_, h := ctx.Size()
			ui.Scroll(ctx).Height(clamp(h*0.5, 200, 400)).Children(func() {
				ui.Column(ctx).Gap(4).Children(func() {
					for i := len(c.records) - 1; i >= 0; i-- {
						ui.Text(ctx, c.records[i]).FontSize(14).TextColor(white).SingleLine()
					}
				})
			})
			ui.Row(ctx).FillWidth().Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(ctx, "清空").Padding(8, 16).Radius(10).
					Material(glass.Glass{Clarity: 0.4}).
					TextColor(white).Clicked() {
					c.records = nil
					c.lastExpr = ""
				}
				if ui.Button(ctx, "关闭").Padding(8, 16).Radius(10).
					Material(glass.Glass{Tint: ui.Hex("#ff9f0a"), Clarity: 0.5, Interactive: true}).
					TextColor(white).Clicked() {
					c.showRecords = false
				}
			})
		})
	})
}

func main() {
	calc := NewCalc()

	mygo.App.WhenReady(func() {
		// 响应式边界：根据主显示器工作区动态计算，不写死数值。
		wa := mygo.Screen.PrimaryDisplay().WorkArea
		// 最小尺寸：保证按钮与显示屏完整显示（按最小按钮 36px × 5 行估算）
		minW := clampInt(wa.Width*18/100, 280, 400)
		minH := clampInt(wa.Height*35/100, 520, 820)
		// 最大尺寸：不超过工作区的合理比例，避免窗口过大
		maxW := clampInt(wa.Width*45/100, 460, 720)
		maxH := clampInt(wa.Height*80/100, 760, 1200)
		// 确保 min <= max（极小屏幕下收敛）
		if minW > maxW {
			minW = maxW
		}
		if minH > maxH {
			minH = maxH
		}
		// 初始尺寸按 430×850 的窄高比例（宽:高 ≈ 0.506），再等比例缩小 1.2 倍
		initH := clampInt(wa.Height*69/120, minH, maxH) // 69%/1.2 ≈ 57.5%
		initW := clampInt(int(float32(initH)*0.506), minW, maxW)

		mygo.NewWindow(mygo.WindowOptions{
			Title:      "计算器 - MyGo",
			Width:      initW,
			Height:     initH,
			MinWidth:   minW,
			MinHeight:  minH,
			MaxWidth:   maxW,
			MaxHeight:  maxH,
			Content:    ui.View(calc.view),
		})
	})

	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}
