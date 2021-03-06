/*
 Copyright 2020 The GoPlus Authors (goplus.org)

 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package bytecode

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/qiniu/x/log"
)

// -----------------------------------------------------------------------------

func init() {
	log.SetOutputLevel(log.Ldebug)
	if opCallGoFunc != SymbolFunc || opCallGoFuncv != SymbolFuncv {
		panic("opCallGoFunc != SymbolFunc || opCallGoFuncv != SymbolFuncv")
	}
}

func Strcat(a, b string) string {
	return a + b
}

func execStrcat(arity int, p *Context) {
	args := p.GetArgs(2)
	ret := Strcat(args[0].(string), args[1].(string))
	p.Ret(2, ret)
}

func execPrintln(arity int, p *Context) {
	args := p.GetArgs(arity)
	n, err := fmt.Println(args...)
	p.Ret(arity, n, err)
}

func execSprint(arity int, p *Context) {
	args := p.GetArgs(arity)
	s := fmt.Sprint(args...)
	p.Ret(arity, s)
}

func execSprintf(arity int, p *Context) {
	args := p.GetArgs(arity)
	s := fmt.Sprintf(args[0].(string), args[1:]...)
	p.Ret(arity, s)
}

func execErrorf(arity int, p *Context) {
	args := p.GetArgs(arity)
	s := fmt.Errorf(args[0].(string), args[1:]...)
	p.Ret(arity, s)
}

// I is a Go package instance.
var I = NewGoPackage("foo")

func init() {
	I.RegisterFuncvs(
		I.Funcv("Sprint", fmt.Sprint, execSprint),
		I.Funcv("Sprintf", fmt.Sprintf, execSprintf),
		I.Funcv("Errorf", fmt.Errorf, execErrorf),
		I.Funcv("Println", fmt.Println, execPrintln),
	)
	I.RegisterFuncs(
		I.Func("strcat", Strcat, execStrcat),
	)
	I.RegisterVars(
		I.Var("x", new(int)),
		I.Var("y", new(int)),
	)
	I.RegisterConsts(
		I.Const("true", reflect.Bool, true),
		I.Const("false", reflect.Bool, false),
		I.Const("nil", ConstUnboundPtr, nil),
	)
	I.RegisterTypes(
		I.Rtype(reflect.TypeOf((*Context)(nil))),
		I.Rtype(reflect.TypeOf((*Code)(nil))),
		I.Rtype(reflect.TypeOf((*Stack)(nil))),
		I.Type("rune", TyRune),
	)
	_ = I.PkgPath()
}

func TestVarAndConst(t *testing.T) {
	if ci, ok := I.FindConst("true"); !ok || ci.Value != true {
		t.Fatal("FindConst failed:", ci.Value)
	}
	if ci, ok := I.FindConst("nil"); !ok || ci.Kind != ConstUnboundPtr {
		t.Fatal("FindConst failed:", ci.Kind)
	}
	if addr, ok := I.FindVar("x"); !ok || addr != 0 {
		t.Fatal("FindVar failed:", addr)
	}
}

func TestGoVar(t *testing.T) {
	sprint, ok := I.FindFuncv("Sprint")
	strcat, ok2 := I.FindFunc("strcat")
	if !ok || !ok2 {
		t.Fatal("FindFunc failed: Sprintf/strcat")
	}

	pkg := NewGoPackage("pkg")

	pkg.RegisterVars(
		I.Var("x", new(string)),
		I.Var("y", new(string)),
		I.Var("i", new(int)),
		I.Var("j", new(int)),
	)

	x, ok := pkg.FindVar("x")
	if !ok {
		t.Fatal("FindVar failed:", x)
	}
	y, ok := pkg.FindVar("y")
	if !ok {
		t.Fatal("FindVar failed:", y)
	}
	i, ok := pkg.FindVar("i")
	if !ok {
		t.Fatal("FindVar failed:", i)
	}
	j, ok := pkg.FindVar("j")
	if !ok {
		t.Fatal("FindVar failed:", j)
	}

	b := newBuilder()
	code := b.
		Push(5).
		Push("32").
		CallGoFuncv(sprint, 2, 2).
		StoreGoVar(x). // x = sprint(5, "32")
		Push("78").
		LoadGoVar(x).
		CallGoFunc(strcat, 2).
		StoreGoVar(y). // y = strcat("78", x)
		Push(5).
		StoreGoVar(i).
		LoadGoVar(i).
		AddrGoVar(j).
		AddrOp(Int, OpAssign).
		Push(5).
		AddrGoVar(j).
		AddrOp(Int, OpAddAssign).
		Resolve()

	ctx := NewContext(code)
	ctx.Exec(0, code.Len())

	if v := reflect.ValueOf(govars[x].Addr).Elem().String(); v != "532" {
		t.Fatal("x != 532, x = ", v)
	}
	if v := reflect.ValueOf(govars[y].Addr).Elem().String(); v != "78532" {
		t.Fatal("y != 78532, x =", v)
	}
	if v := reflect.ValueOf(govars[i].Addr).Elem().Int(); v != 5 {
		t.Fatal("i !=5, x =", i)
	}
	if v := reflect.ValueOf(govars[j].Addr).Elem().Int(); v != 10 {
		t.Fatal("j !=10, x =", j)
	}
}

type testPoint struct {
	X int
	Y int
}

type testInfo struct {
	Info string
}

type testRect struct {
	testInfo
	Pt1 testPoint
	Pt2 *testPoint
}

var gRect *testRect

func init() {
	gRect = &testRect{}
	gRect.Pt2 = &testPoint{}
}

func getTestRect() *testRect {
	return gRect
}

func execTestRect(_ int, p *Context) {
	p.Ret(0, getTestRect())
}

func TestGoField(t *testing.T) {
	sprint, ok := I.FindFuncv("Sprint")

	pkg := NewGoPackage("pkg_field")

	rc := testRect{}
	rc.Info = "Info"
	rc.Pt1 = testPoint{10, 20}
	rc.Pt2 = &testPoint{30, 40}

	rc2 := testRect{}
	rc2.Pt2 = &testPoint{}

	pkg.RegisterVars(
		pkg.Var("Rect", &rc),
		pkg.Var("Rect2", &rc2),
	)
	pkg.RegisterFuncs(
		pkg.Func("GetRect", getTestRect, execTestRect),
	)

	x, ok := pkg.FindVar("Rect")
	if !ok {
		t.Fatal("FindVar failed:", x)
	}
	x2, ok := pkg.FindVar("Rect2")
	if !ok {
		t.Fatal("FindVar failed:", x2)
	}
	fnTestRect, ok := pkg.FindFunc("GetRect")
	if !ok {
		t.Fatal("FindFunc failed:", fnTestRect)
	}

	it := reflect.TypeOf(rc)
	it2 := reflect.TypeOf(gRect)

	y := NewVar(it, "y")
	b := newBuilder()

	code := b.
		DefineVar(y). // y
		LoadGoVar(x2).
		StoreVar(y). // y = pkg_field.Rect2
		Push("hello").
		StoreField(x, []int{0, 0}). // pkg_field.Rect.Info = "hello"
		Push(-1).
		StoreField(x, []int{1, 0}). // pkg_field.Rect.Pt1.X = -1
		Push(-2).
		StoreField(x, []int{2, 1}). // pkg_field.Rect.Pt2.Y = -2
		Push("world").
		StoreField(y, []int{0, 0}). // y.Info = "world"
		Push(-10).
		StoreField(y, []int{1, 0}). // y.Pt1.X = -10
		Push(-20).
		StoreField(y, []int{2, 1}). // y.Pt2.Y = -20
		Push("next").
		CallGoFunc(fnTestRect, 0).
		StoreField(it2, []int{0, 0}). // pkg_field.GetRect().Info = "next"
		Push(101).
		CallGoFunc(fnTestRect, 0).
		StoreField(it2, []int{1, 0}). // pkg_field.GetRect().Pt1.X = 101
		Push(102).
		CallGoFunc(fnTestRect, 0).
		StoreField(it2, []int{2, 1}). // pkg_field.GetRect().Pt2.Y = 102
		LoadVar(y).
		StoreGoVar(x2). // pkg_field.Rect2 = y
		LoadField(x, []int{0, 0}).
		LoadField(x, []int{2, 1}).
		LoadField(y, []int{0, 0}).
		LoadField(y, []int{2, 1}).
		CallGoFunc(fnTestRect, 0).
		LoadField(it2, []int{0, 0}).
		CallGoFunc(fnTestRect, 0).
		LoadField(it2, []int{2, 1}).
		AddrField(x, []int{1}).
		AddrField(y, []int{1}).
		CallGoFunc(fnTestRect, 0).
		AddrField(it2, []int{1}).
		CallGoFuncv(sprint, 9, 9). // print(pkg_field.Info,pkg_field.Pt2.Y,y.Info,y.Pt2.Y,pkg_field.GetRect().Info,pkg_field.GetRect().Pt2.Y)
		Resolve()

	ctx := NewContext(code)
	ctx.Exec(0, code.Len())

	if rc.Info != "hello" || rc.Pt1.X != -1 || rc.Pt2.Y != -2 {
		t.Fatal("Rect", rc)
	}
	if rc2.Info != "world" || rc2.Pt1.X != -10 || rc2.Pt2.Y != -20 {
		t.Fatal("Rect2", rc2)
	}
	if gRect.Info != "next" || gRect.Pt1.X != 101 || gRect.Pt2.Y != 102 {
		t.Fatal("g_Rect", gRect)
	}
	if v := ctx.Get(-1); v != "hello-2world-20next102 &{-1 20} &{-10 0} &{101 0}" {
		t.Fatal("LoadField", v)
	}
}

func TestSprint(t *testing.T) {
	sprint, ok := I.FindFuncv("Sprint")
	if !ok {
		t.Fatal("FindFuncv failed: Sprint")
	}

	code := newBuilder().
		Push(5).
		Push("32").
		CallGoFuncv(sprint, 2, 2).
		Resolve()

	ctx := NewContext(code)
	ctx.Exec(0, code.Len())
	if v := checkPop(ctx); v != "532" {
		t.Fatal("5 `32` sprint != 532, ret =", v)
	}
}

func TestSprintf(t *testing.T) {
	sprintf, ok := I.FindFuncv("Sprintf")
	strcat, ok2 := I.FindFunc("strcat")
	if !ok || !ok2 {
		t.Fatal("FindFunc failed: Sprintf/strcat")
	}
	_ = defaultImpl.GetGoFuncType(strcat)
	_ = defaultImpl.GetGoFuncvType(sprintf)

	code := newBuilder().
		Push("Hello, %v, %d, %s").
		Push(1.3).
		Push(1).
		Push("x").
		Push("sw").
		CallGoFunc(strcat, 2).
		CallGoFuncv(sprintf, 4, 4).
		Resolve()

	ctx := NewContext(code)
	ctx.Exec(0, code.Len())
	if v := checkPop(ctx); v != "Hello, 1.3, 1, xsw" {
		t.Fatal("format 1.3 1 `x` `sw` strcat sprintf != `Hello, 1.3, 1, xsw`, ret =", v)
	}
}

func TestLargeArity(t *testing.T) {
	sprint, kind, ok := defaultImpl.FindGoPackage("foo").Find("Sprint")
	if !ok || kind != SymbolFuncv {
		t.Fatal("Find failed: Sprint")
	}

	b := newBuilder()
	ret := ""
	for i := 0; i < bitsFuncvArityMax+1; i++ {
		b.Push("32")
		ret += "32"
	}
	code := b.
		CallGoFuncv(GoFuncvAddr(sprint), bitsFuncvArityMax+1, bitsFuncvArityMax+1).
		Resolve()

	ctx := NewContext(code)
	ctx.Exec(0, code.Len())
	if v := checkPop(ctx); v != ret {
		t.Fatal("32 times(1024) sprint != `32` times(1024), ret =", v)
	}
}

func TestType(t *testing.T) {
	typ, ok := I.FindType("Context")
	if !ok {
		t.Fatal("FindType failed: Context not found")
	}
	fmt.Println(typ)

	typ, ok = FindGoPackage("foo").FindType("rune")
	if !ok || typ != TyRune {
		t.Fatal("FindType failed: rune not found")
	}
	fmt.Println(typ)
}
