// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package commandline

import (
	"errors"
	"fmt"
	"testing"
)

func TestCommands(t *testing.T) {
	cl := New()
	root := cl.MustAddCommand("", "", nil)
	if _, err := cl.AddCommand("", "", nil); err == nil {
		t.Fatal("Failed detecting duplicate empty root command.")
	}
	if _, err := root.AddCommand("", "", nil); err == nil {
		t.Fatal("Failed detecting empty command name in non-root command.")
	}
	root.MustAddCommand("foo", "", nil)
	if _, err := root.AddCommand("foo", "", nil); err == nil {
		t.Fatal("Failed detecting duplicate empty root command name.")
	}
	var fooval string
	root.MustAddParam("foo", "", "", true, &fooval)
	if err := cl.Parse([]string{"--foo", "foo"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"--boo"}); err == nil {
		t.Fatal("Failed detecting non-existent empty root command param.")
	}
	if err := cl.Parse([]string{"--boo", "boo"}); err == nil {
		t.Fatal("Failed detecting non-existent empty root command param.")
	}
	if err := cl.Parse([]string{"boo"}); err == nil {
		t.Fatal("failed detecting non-existent command.")
	}
	if err := cl.Parse([]string{}); err == nil {
		t.Fatal("failed detecting no arguments.")
	}
}

func TestVisit(t *testing.T) {
	cl := New()
	var foocount int
	var foo = func(ctx Context) error {
		foocount++
		if ctx.Executed() {
			t.Fatal("Command.Executed() failed.")
		}
		return nil
	}
	var barcount int
	var bar = func(ctx Context) error {
		barcount++
		if ctx.Executed() {
			t.Fatal("Command.Executed() failed.")
		}
		return nil
	}
	var bazcount int
	var baz = func(ctx Context) error {
		bazcount++
		if ctx.Executed() {
			t.Fatal("Command.Executed() failed.")
		}
		return nil
	}
	var batcount int
	var bat = func(ctx Context) error {
		batcount++
		if !ctx.Executed() {
			t.Fatal("Command.Executed() failed.")
		}
		return nil
	}
	cl.MustAddCommand("foo", "", foo).
		MustAddCommand("bar", "", bar).
		MustAddCommand("baz", "", baz).
		MustAddCommand("bat", "", bat)
	if err := cl.Parse([]string{"foo", "bar", "baz", "bat"}); err != nil {
		t.Fatal(err)
	}
	if foocount != 1 || barcount != 1 || bazcount != 1 || batcount != 1 {
		fmt.Println(foocount, barcount, bazcount, batcount)
		t.Fatal("Visit failed.")
	}
}

func TestContextPrefixed(t *testing.T) {
	var bar string
	var foo = func(ctx Context) error {
		if !ctx.Parsed("bar") {
			t.Fatal("Context failed.")
		}
		if ctx.Arg("bar") != "baz" {
			t.Fatal("Context failed.")
		}
		if len(ctx.Args()) != 0 {
			t.Fatal("Context failed.")
		}
		return nil
	}
	cl := New()
	cl.MustAddCommand("foo", "", foo).MustAddParam("bar", "b", "", false, &bar)
	if err := cl.Parse([]string{"foo", "--bar", "baz"}); err != nil {
		t.Fatal(err)
	}
}

func TestContextRaw(t *testing.T) {
	var foo = func(ctx Context) error {
		if !ctx.Parsed("bar") {
			t.Fatal("Context failed.")
		}
		if ctx.Arg("bar") != "bar" {
			t.Fatal("Context failed.")
		}
		if len(ctx.Args()) != 0 {
			t.Fatal("Context failed.")
		}
		return nil
	}
	cl := New()
	cl.MustAddCommand("foo", "", foo).MustAddRawParam("bar", "", false, nil)
	if err := cl.Parse([]string{"foo", "bar"}); err != nil {
		t.Fatal(err)
	}
}

func TestGlobal(t *testing.T) {
	cl := New()
	barCmd := func(params Context) error {
		return nil
	}
	if err := cl.Parse([]string{"--verbose"}); err == nil {
		t.Fatal("failed detecting command not specified.")
	}
	if err := cl.Parse([]string{"verbose"}); err == nil {
		t.Fatal("failed detecting unregistered command.")
	}
	cl.MustAddCommand("", "", nil).MustAddParam("verbose", "v", "", false, nil)
	if err := cl.Parse([]string{"--verbose"}); err != nil {
		t.Fatal(err)
	}
	cl.MustAddCommand("foo", "", nil)
	if err := cl.Parse([]string{"--verbose", "foo"}); err == nil {
		t.Fatal("Failed detecting command with no handler.")
	}
	cl.MustAddCommand("bar", "", barCmd)
	if err := cl.Parse([]string{"--verbose", "bar"}); err != nil {
		t.Fatal(err)
	}
}

func TestNoRegisteredParams(t *testing.T) {
	ErrOK := errors.New("everything is fine")
	fooArgs := []string{"foo"}
	barArgs := []string{"bar", "one", "two", "three"}
	bazArgs := []string{"baz", "one", "two", "three"}
	cmdFoo := func(params Context) error {
		return ErrOK
	}
	cmdBar := func(params Context) error {
		for idx, arg := range params.Args() {
			if barArgs[idx+1] != arg {
				t.Fatal("Unregistered param mode failed.")
			}
		}
		return nil
	}
	cl := New()
	cl.MustAddCommand("foo", "", cmdFoo)
	cl.MustAddCommand("bar", "", cmdBar)
	cl.MustAddCommand("baz", "", nil)
	if err := cl.Parse(fooArgs); err != ErrOK {
		t.Fatal(err)
	}
	if err := cl.Parse(barArgs); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse(bazArgs); err == nil {
		t.Fatal("Failed detecting command not having a handler for raw args.")
	}
}

func TestRegisteredRaw(t *testing.T) {
	cmdFoo := func(params Context) error {
		if params.Arg("bar") != "bar" {
			t.Fatal("RegisteredRaw failed.")
		}
		if params.Arg("baz") != "baz" {
			t.Fatal("RegisteredRaw failed.")
		}
		return nil
	}
	cl := New()
	cmd := cl.MustAddCommand("foo", "", cmdFoo).
		MustAddRawParam("bar", "", true, nil).
		MustAddRawParam("baz", "", false, nil)
	if err := cmd.AddRawParam("boo", "", false, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of optional raw param after optional raw param.")
	}
	if err := cmd.AddRawParam("boo", "", true, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of required raw param after optional raw param.")
	}
	if err := cmd.AddParam("boo", "", "", false, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of optional prefixed param after raw param.")
	}
	if err := cmd.AddParam("boo", "", "", true, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param after raw param.")
	}
	if _, err := cmd.AddCommand("", "", nil); err == nil {
		t.Fatal("Failed detecting invalid registration of a non-root command with an empty name.")
	}
	if _, err := cmd.AddCommand("boo", "", nil); err == nil {
		t.Fatal("Failed detecting invalid registration of a command on a command with defined raw args.")
	}
	if err := cl.Parse([]string{"foo", "bar", "baz"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "bar", "baz", "boo"}); err == nil {
		t.Fatal("Failed detecting extra arguments for registered raw params.")
	}
}

func TestRegisteredRawRequired(t *testing.T) {
	cl := New()
	var barval, bazval string
	cl.MustAddCommand("foo", "", nil).
		MustAddRawParam("bar", "", true, &barval).
		MustAddRawParam("baz", "", true, &bazval)
	if err := cl.Parse([]string{"foo", "bar", "baz"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "bar"}); err == nil {
		t.Fatal("Failed detecting required params not specified.")
	}
	if err := cl.Parse([]string{"foo", "bar", "baz", "bit"}); err == nil {
		t.Fatal("Failed detecting extra raw params.")
	}
}

func TestPrefixed(t *testing.T) {
	barv := ""
	cmdFoo := func(params Context) error {
		if params.Arg("bar") != "bar" {
			t.Fatal("TestPrefixed failed.")
		}
		if !params.Parsed("baz") {
			t.Fatal("TestPrefixed failed.")
		}
		return nil
	}
	cl := New()
	cmd := cl.MustAddCommand("foo", "", cmdFoo)
	if err := cmd.AddParam("boo", "", "", true, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param without valid value.")
	}
	if err := cmd.AddParam("boo", "", "", true, 1337); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param with invalid value.")
	}
	cmd.MustAddParam("bar", "r", "", true, &barv).
		MustAddParam("baz", "z", "", false, nil).
		MustAddParam("bat", "t", "", false, nil)
	if err := cmd.AddParam("boo", "", "", true, &barv); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param after optional prefixed param.")
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "-r", "bar", "-z"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "-h"}); err == nil {
		t.Fatal("Failed detecting non-existent short param.")
	}
	if err := cl.Parse([]string{"foo", "--bit", "bit"}); err == nil {
		t.Fatal("Failed detecting non-existent parameters.")
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "--bit"}); err == nil {
		t.Fatal("Failed detecting extra params.")
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "--bit", "bit"}); err == nil {
		t.Fatal("Failed detecting extra params.")
	}
	if err := cl.Parse([]string{"foo", "--baz"}); err == nil {
		t.Fatal("Failed detecting required params not specified.")
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "boo"}); err == nil {
		t.Fatal("Failed detecting required params not specified.")
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "--bat", "boo"}); err == nil {
		t.Fatal("Failed detecting extra arguments.")
	}
}

func TestPrefixedRequired(t *testing.T) {
	cl := New()
	var barval, bazval string
	cl.MustAddCommand("foo", "", nil).
		MustAddParam("bar", "", "", true, &barval).
		MustAddParam("baz", "", "", true, &bazval)
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "baz"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz"}); err == nil {
		t.Fatal("Failed detecting param argument not specified.")
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar"}); err == nil {
		t.Fatal("Failed detecting command not specified.")
	}
}

func TestPrefixedShort(t *testing.T) {
	cl := New()
	var valFilip string
	root := cl.MustAddCommand("foo", "", nil).
		MustAddParam("alice", "a", "", false, nil).
		MustAddParam("buick", "b", "", false, nil).
		MustAddParam("cecil", "c", "", false, nil).
		MustAddParam("david", "d", "", false, nil).
		MustAddParam("emily", "e", "", false, nil).
		MustAddParam("filip", "f", "", false, &valFilip)
	if err := root.AddParam("alice", "", "", false, nil); err == nil {
		t.Fatal("Failed detecting duplicate long name.")
	}
	if err := root.AddParam("agnes", "a", "", false, nil); err == nil {
		t.Fatal("Failed detecting duplicate short name.")
	}
	if err := root.AddParam("agnes", "agnes", "", false, nil); err == nil {
		t.Fatal("Failed detecting invalid short name.")
	}
	if err := cl.Parse([]string{"foo", "-abcde"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "--alice", "--cecil", "--emily", "-bd"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "-ab", "--cecil", "-de"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "--alice", "-bcd", "--emily"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "-xyz"}); err == nil {
		t.Fatal("Failed detecting invalid short name.")
	}
	if err := cl.Parse([]string{"foo", "-abcde", "-f", "filip"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "-abcdef", "filip"}); err == nil {
		t.Fatal("Failed detecting short param with required value being combined.")
	}
}

func TestCombined(t *testing.T) {
	cl := New()
	var valBar, valBit string
	cl.MustAddCommand("foo", "", nil).
		MustAddParam("bar", "", "", true, &valBar).
		MustAddParam("baz", "", "", false, nil).
		MustAddRawParam("bit", "", true, &valBit).
		MustAddRawParam("bot", "", false, nil)
	if err := cl.Parse([]string{"foo", "--bar", "bar", "bit"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "bit", "bot"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar"}); err == nil {
		t.Fatal("Failed detecting missing required argument to raw param with required value.")
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "bit"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "bit", "bot"}); err != nil {
		t.Fatal(err)
	}
	if err := cl.Parse([]string{"foo", "--bar", "bar", "--baz", "bit", "bot", "boo"}); err == nil {
		t.Fatal("Failed detecting extra argument to optional prefixed long param without required value.")
	}
}

func TestCombinedRequired(t *testing.T) {
	cl := New()
	cl.MustAddCommand("foo", "", nil).
		MustAddCommand("bar", "", nil).
		MustAddParam("baz", "", "", false, nil).
		MustAddRawParam("bat", "", true, nil)
	if err := cl.Parse([]string{"foo", "bar"}); err == nil {
		t.Fatal("Failed detecting required raw param")
	}
	if err := cl.Parse([]string{"foo", "bar", "bat"}); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkParser(b *testing.B) {
	cl := New()
	var barVal string
	var cmdfunc = func(ctx Context) error {
		return nil
	}
	cl.MustAddCommand("", "", nil).MustAddParam("verbose", "v", "Verbose output.", false, nil)
	cl.MustAddCommand("foo", "Do the foo.", nil).MustAddParam("bar", "r", "Enable bar.", true, &barVal).
		MustAddCommand("baz", "Do the baz.", cmdfunc).MustAddRawParam("bat", "Enable bat.", false, nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cl.Parse([]string{"--verbose", "foo", "--bar", "bar", "baz", "bat"})
	}
}

func ExampleParser() {
	cl := New()
	var barVal string
	var cmdfunc = func(ctx Context) error {
		return nil
	}
	cl.MustAddCommand("", "", nil).MustAddParam("verbose", "v", "Verbose output.", false, nil)
	cl.MustAddCommand("foo", "Do the foo.", nil).MustAddParam("bar", "r", "Enable bar.", true, &barVal).
		MustAddCommand("baz", "Do the baz.", cmdfunc).MustAddRawParam("bat", "Enable bat.", false, nil)
	if err := cl.Parse([]string{"--verbose", "foo", "--bar", "bar", "baz", "bat"}); err != nil {
		panic(err)
	}
	fmt.Println(cl.Print())
	// Output:[--verbose]     -v      Verbose output.

	// foo     Do the foo.
	// <--bar> -r      (string)        Enable bar.

	// baz     Do the baz.
	//		[bat]   Enable bat.
}
