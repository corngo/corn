package infra

import (
	"fmt"
	"strings"

	. "github.com/infrago/base"
)

var (
	OK      = Result(0, "ok", "成功")
	Fail    = Result(1, "fail", "失败")
	Retry   = Result(2, "retry", "重试")
	Invalid = Result(3, "invalid", "无效请求或数据")
	Nothing = Result(4, "nothing", "无效对象")

	Unsigned = Result(5, "unsigned", "无权访问")
	Unauthed = Result(6, "unauthed", "无权访问")

	varEmpty = Result(7, "varempty", "%s不可为空")
	varError = Result(8, "varerrpr", "%s无效")
)

type (
	result struct {
		// code 状态码
		// 0 表示成功，其它表示失败
		code int
		// state 对应的状态
		state string
		//携带的参数
		args []Any
		//重试标记
		//专为队列模块
		retry bool
	}
)

// OK 表示Res是否成功
func (res *result) OK() bool {
	if res == nil {
		return true
	}
	return res.code == 0
}

// Fail 表示Res是否失败
func (res *result) Fail() bool {
	return !res.OK()
}

// Code 返回Res的状态码
func (res *result) Code() int {
	return res.code
}

// State 返回Res的信息
func (res *result) State() string {
	return res.state
}

// Args 返回Res携带的参数
func (res *result) Args() []Any {
	return res.args
}

// With 使用当前Res加上参数生成一个新的Res并返回
// 因为result都是预先定义好的，所以如果直接修改args，会修改本来已经定义好的result
func (res *result) With(args ...Any) Res {
	if len(args) > 0 {
		return &result{res.code, res.state, args, false}
	}
	return res
}

// Error 返回Res的信息以符合error接口的定义
func (res *result) Error() string {
	return res.String()
}

// Retry
func (res *result) String() string {
	if res == nil {
		return ""
	}

	text := String(DEFAULT, res.state)

	if res.args != nil && len(res.args) > 0 {
		ccc := strings.Count(text, "%") - strings.Count(text, "%%")
		if ccc == len(res.args) {
			return fmt.Sprintf(text, res.args...)
		}
	}
	return text
}

func newResult(code int, text string, args ...Any) Res {
	return &result{code, text, args, false}
}
func codeResult(code int, args ...Any) Res {
	return &result{code, "", args, false}
}
func textResult(text string, args ...Any) Res {
	return &result{-1, text, args, false}
}
func errorResult(err error) Res {
	return &result{-1, err.Error(), []Any{}, false}
}

// Result 定义一个result，并自动注册state
// state 表示状态key
// text 表示状态对应的默认文案
func Result(code int, state string, text string) Res {
	//自动注册状态和字串
	infraBasic.State(state, State(code))
	infraBasic.Strings(DEFAULT, Strings{state: text})

	// result只携带state，而不携带string
	// 具体的string需要配置context拿到lang之后生成
	// 而实现多语言的状态反馈
	return newResult(code, state)
}

// func NewResult(code int, text string, args ...Any) Res {
// 	return &result{code, text, args}
// }

//-----------------------------------------------

type (
	resultGroup struct {
		name string
		base int
	}
)

//------------ resultGroup ----------------

func (this *resultGroup) Name() string {
	return this.name
}
func (this *resultGroup) Result(ok bool, state string, text string) Res {
	code := 0
	if ok == false {
		code = this.base
		this.base++
	}

	if this.name != "" && !strings.HasPrefix(state, this.name+".") {
		state = this.name + "." + state
	}
	return Result(code, state, text)
}

func ResultGroup(name string, bases ...int) *resultGroup {

	base := 1000
	if len(bases) > 0 {
		base = bases[0]
	}
	return &resultGroup{name, base}
}
