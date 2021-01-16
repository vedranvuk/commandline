// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package commandline

import (
	"errors"
	"fmt"
	"testing"
)

// Test parser's command line argument detection.
func TestParserNext(t *testing.T) {
	var cl = NewState()
	var arg string
	var kind Argument
	cl.arguments = []string{}
	if arg, kind = cl.Next(); arg != "" || kind != NoArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{""}
	if arg, kind = cl.Next(); arg != "" || kind != NoArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{"-"}
	if arg, kind = cl.Next(); arg != "" || kind != InvalidArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{"--"}
	if arg, kind = cl.Next(); arg != "" || kind != InvalidArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{"---"}
	if arg, kind = cl.Next(); arg != "" || kind != InvalidArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{"foo"}
	if arg, kind = cl.Next(); arg != "foo" || kind != TextArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{"-f"}
	if arg, kind = cl.Next(); arg != "f" || kind != ShortArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{"--foo"}
	if arg, kind = cl.Next(); arg != "foo" || kind != LongArgument {
		t.Fatal("Error parsing argument.")
	}
	cl.arguments = []string{"-foo"}
	if arg, kind = cl.Next(); arg != "foo" || kind != CombinedArgument {
		t.Fatal("Error parsing argument.")
	}
}

// Test command registrations and execution.
func TestCommands(t *testing.T) {
	var cmdfunc = func(ctx Context) error {
		return nil
	}
	var state = NewState()
	var err error
	// Register empty root command.
	var command = state.MustAddCommand("", "", nil)
	// Cannot re-register empty root command.
	if _, err = state.AddCommand("", "", nil); err == nil {
		t.Fatal("Failed detecting duplicate empty root command.")
	}
	var fooval string
	// Register a "foo" param with required arg on empty root command.
	command.MustAddParam("foo", "", "", true, &fooval)
	// "--foo" parses "foo" to fooval for empty root command.
	if err = state.Parse([]string{"--foo", "foo"}); err != nil {
		t.Fatal(err)
	}
	// "--boo" is not a registered empty root command param.
	if err := state.Parse([]string{"--boo"}); err == nil {
		t.Fatal("Failed detecting non-existent empty root command param.")
	}
	// again, but with an argument.
	if err := state.Parse([]string{"--boo", "boo"}); err == nil {
		t.Fatal("Failed detecting non-existent empty root command param.")
	}
	// Register "foo" command in root.
	command = state.MustAddCommand("foo", "", cmdfunc)
	// No duplicate command names in root.
	if _, err = state.AddCommand("foo", "", nil); err == nil {
		t.Fatal("Failed detecting duplicate command name.")
	}
	// "foo" is a registered command.
	if err = state.Parse([]string{"foo"}); err != nil {
		t.Fatal("Failed parsing command.")
	}
	// Add "bar" sub-command.
	command.MustAddCommand("bar", "", cmdfunc)
	// No duplicate command names in subs.
	if _, err = command.AddCommand("bar", "", nil); err == nil {
		t.Fatal("Failed detecting duplicate command name.")
	}
	// "foo" has "bar" sub-command.
	if err = state.Parse([]string{"foo", "bar"}); err != nil {
		t.Fatal("Failed parsing commands.")
	}
	// "boo" is not a registered command.
	if err = state.Parse([]string{"boo"}); err == nil {
		t.Fatal("Failed detecting non-existent command.")
	}
	// "boo" will be passed as raw argument to "foo" but "foo" is not a raw
	// command.
	if err = state.Parse([]string{"foo", "boo"}); !errors.Is(err, ErrExtraArguments) {
		t.Fatal(err)
	}
}

// Handlers of commands detected on command line are executed as parsed.
// Only the last command in execution chain will have context.Executed true.
// Each handler in chain must be executed only once.
func TestHandlerVisit(t *testing.T) {
	var foocount int
	var foo = func(ctx Context) error {
		foocount++
		if ctx.Executed() {
			t.Fatal("Command in chain marked as executed.")
		}
		return nil
	}
	var barcount int
	var bar = func(ctx Context) error {
		barcount++
		if ctx.Executed() {
			t.Fatal("Command in chain marked as executed.")
		}
		return nil
	}
	var bazcount int
	var baz = func(ctx Context) error {
		bazcount++
		if ctx.Executed() {
			t.Fatal("Command in chain marked as executed.")
		}
		return nil
	}
	var batcount int
	var bat = func(ctx Context) error {
		batcount++
		if !ctx.Executed() {
			t.Fatal("Command in chain not marked as executed.")
		}
		return nil
	}
	var cl = NewState()
	cl.MustAddCommand("foo", "", foo).
		MustAddCommand("bar", "", bar).
		MustAddCommand("baz", "", baz).
		MustAddCommand("bat", "", bat)
	var err error
	if err = cl.Parse([]string{"foo", "bar", "baz", "bat"}); err != nil {
		t.Fatal(err)
	}
	if foocount != 1 || barcount != 1 || bazcount != 1 || batcount != 1 {
		fmt.Println(foocount, barcount, bazcount, batcount)
		t.Fatal("Multiple handler visitations.")
	}
}

// Handlers of commands can return a non-nil error to stop further parsing.
func TestHandlerErrorPropagation(t *testing.T) {
	var handlererror = errors.New("propagated error")
	var foo = func(ctx Context) error {
		// Returns nil, parsing should continue.
		return nil
	}
	var bar = func(ctx Context) error {
		// Returns an error, parsing should stop.
		return handlererror
	}
	var baz = func(ctx Context) error {
		// This handler should not be reached.
		t.Fatal("Executed handler of a sub command after an error in a chain.")
		return nil
	}
	var cl = NewState()
	cl.MustAddCommand("foo", "", foo).
		MustAddCommand("bar", "", bar).
		MustAddCommand("baz", "", baz)
	var err error
	if err = cl.Parse([]string{"foo", "bar", "baz"}); err != handlererror {
		t.Fatal("Handler error propagation failed.")
	}
}

// Prefixed params of handler's command can be retrieved
// by long name from Context.
func TestPrefixedParamFromContext(t *testing.T) {
	var bar string
	var foo = func(ctx Context) error {
		if !ctx.Parsed("bar") {
			t.Fatal("Prefixed param not marked as parsed.")
		}
		if ctx.Value("bar") != "bar" {
			t.Fatal("Unexpected prefixed param value.")
		}
		if len(ctx.Arguments()) != 0 {
			t.Fatal("Raw args must be accessible only when no params were registered.")
		}
		return nil
	}
	var cl = NewState()
	var err error
	cl.MustAddCommand("foo", "", foo).MustAddParam("bar", "b", "", false, &bar)
	if err = cl.Parse([]string{"foo", "--bar", "bar"}); err != nil {
		t.Fatal(err)
	}
}

// Raw params of handler's command can be retrieved
// by long name from Context.
func TestRawParamFromContext(t *testing.T) {
	var foo = func(ctx Context) error {
		if !ctx.Parsed("bar") {
			t.Fatal("Raw param not marked as parsed.")
		}
		if ctx.Value("bar") != "bar" {
			t.Fatal("Unexpected raw param value.")
		}
		if len(ctx.Arguments()) != 0 {
			t.Fatal("Raw args must be accessible only when no params were registered.")
		}
		return nil
	}
	var cl = NewState()
	var err error
	cl.MustAddCommand("foo", "", foo).MustAddRawParam("bar", "", false, nil)
	if err = cl.Parse([]string{"foo", "bar"}); err != nil {
		t.Fatal(err)
	}
}

// Commands with no registered params can read raw arguments following
// command via Context.
func TestUnregisteredParamFromContext(t *testing.T) {
	var foo = func(ctx Context) error {
		var a = ctx.Arguments()
		if a[0] != "1" || a[1] != "2" || a[2] != "3" {
			t.Fatal("Unexpected raw arguments.")
		}
		return nil
	}
	var cl = NewState()
	var err error
	cl.MustAddRawCommand("foo", "", foo)
	if err = cl.Parse([]string{"foo", "1", "2", "3"}); err != nil {
		t.Fatal(err)
	}
}

// Test prefixed parameters.
func TestPrefixed(t *testing.T) {
	var barv string
	var foo = func(ctx Context) error {
		if ctx.Value("bar") != "bar" {
			t.Fatal("TestPrefixed failed.")
		}
		if !ctx.Parsed("baz") {
			t.Fatal("TestPrefixed failed.")
		}
		return nil
	}
	var cl = NewState()
	var err error
	// Register "foo" command at root.
	var cmd = cl.MustAddCommand("foo", "", foo)
	// Required prefixed params need a pointer to go value to parse into.
	if err = cmd.AddParam("boo", "", "", true, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param without valid value.")
	}
	// Value must be a pointer to Go value.
	if err = cmd.AddParam("boo", "", "", true, 1337); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param with invalid value.")
	}
	// Add required param writing to barv.
	cmd.MustAddParam("bar", "r", "", true, &barv).
		// Add optional baz param.
		MustAddParam("baz", "z", "", false, nil).
		// Add optional bat param.
		MustAddParam("bat", "t", "", false, nil)
	// Duplicate log param name.
	if err = cmd.AddParam("bar", "", "", false, nil); err == nil {
		t.Fatal("Failed detecting duplicate long param name.")
	}
	// Duplicate log param name.
	if err = cmd.AddParam("bit", "r", "", false, nil); err == nil {
		t.Fatal("Failed detecting duplicate short param name.")
	}
	// Must parse ok.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "--baz", "--bat"}); err != nil {
		t.Fatal(err)
	}
	// No such param.
	if err = cl.Parse([]string{"foo", "-h"}); err == nil {
		t.Fatal("Failed detecting non-existent short param.")
	}
	// No such param.
	if err = cl.Parse([]string{"foo", "--bit", "bit"}); err == nil {
		t.Fatal("Failed detecting non-existent parameters.")
	}
	// Non existent param after existing.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "--baz", "--bit"}); err == nil {
		t.Fatal("Failed detecting extra params.")
	}
	// Non existent param after existing with argument.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "--baz", "--bit", "bit"}); err == nil {
		t.Fatal("Failed detecting extra params.")
	}
	// "bar" must be specified.
	if err = cl.Parse([]string{"foo", "--baz"}); err == nil {
		t.Fatal("Failed detecting required params not specified.")
	}
	// "--baz" takes no args and "boo" is not registered as param or subcommand.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "--baz", "boo"}); err == nil {
		t.Fatal("Failed detecting extra params.")
	}
	// "--bat" takes no aprams and "boo" is not registered as param or subcommand.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "--baz", "--bat", "boo"}); err == nil {
		t.Fatal("Failed detecting extra arguments.")
	}
	// Params cannot be specified multiple times.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "--bar", "bar"}); err == nil {
		t.Fatal("Failed detecting duplicate paramaters.")
	}
}

// Test prefixed short combined params.
func TestPrefixedCombined(t *testing.T) {
	var cl = NewState()
	var err error
	var filip string
	var root = cl.MustAddCommand("foo", "", nil).
		MustAddParam("alice", "a", "", false, nil).
		MustAddParam("buick", "b", "", false, nil).
		MustAddParam("cecil", "c", "", false, nil).
		MustAddParam("david", "d", "", false, nil).
		MustAddParam("emily", "e", "", false, nil).
		MustAddParam("filip", "f", "", false, &filip)
	// No duplicate long param names.
	if err = root.AddParam("alice", "", "", false, nil); err == nil {
		t.Fatal("Failed detecting duplicate long name.")
	}
	// No duplicate short param names.
	if err = root.AddParam("agnes", "a", "", false, nil); err == nil {
		t.Fatal("Failed detecting duplicate short name.")
	}
	// Short names must be one char.
	if err = root.AddParam("agnes", "agnes", "", false, nil); err == nil {
		t.Fatal("Failed detecting invalid short name.")
	}
	// Parse combined params ok.
	if err = cl.Parse([]string{"foo", "-abcde"}); err != nil {
		t.Fatal(err)
	}
	// Parse mixed long prefixed and combined short prefixed params ok.
	if err = cl.Parse([]string{"foo", "--alice", "--cecil", "--emily", "-bd"}); err != nil {
		t.Fatal(err)
	}
	// Parse mixed short combined params and long required.
	if err = cl.Parse([]string{"foo", "-abcde", "-f", "filip"}); err != nil {
		t.Fatal(err)
	}
	// Invalid short name in combined short params.
	if err = cl.Parse([]string{"foo", "-abXde"}); err == nil {
		t.Fatal("Failed detecting invalid short name in combined params.")
	}
	// Cannot combine required params into combined short params.
	if err = cl.Parse([]string{"foo", "-abcdef", "filip"}); err == nil {
		t.Fatal("Failed detecting short param with required value being combined.")
	}
	// Parse combined params ok.
	if err = cl.Parse([]string{"foo", "-aa"}); err == nil {
		t.Fatal("Failed detecting duplicate short parameters.")
	}
	// Parse combined params ok.
	if err = cl.Parse([]string{"foo", "-a", "-a"}); err == nil {
		t.Fatal("Failed detecting duplicate short parameters.")
	}
}

// Test registered raw params.
func TestRegisteredRaw(t *testing.T) {
	var foo = func(ctx Context) error {
		if ctx.Value("bar") != "bar" {
			t.Fatal("Unexpected parameter argument value.")
		}
		if ctx.Value("baz") != "baz" {
			t.Fatal("Unexpected parameter argument value.")
		}
		return nil
	}
	var cl = NewState()
	// Register "foo" command.
	var cmd = cl.MustAddCommand("foo", "", foo).
		// Register required "bar" raw parameter.
		MustAddRawParam("bar", "", true, nil).
		// Register optional "baz" raw parameter.
		MustAddRawParam("baz", "", false, nil)
	var err error
	// No prefixed optional parameters after raw parameters.
	if err = cmd.AddRawParam("--boo", "", false, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of required raw param after optional raw param.")
	}
	var boo string
	// No prefixed required parameters after raw parameters.
	if err = cmd.AddRawParam("--boo", "", true, &boo); err == nil {
		t.Fatal("Failed detecting invalid registration of required raw param after optional raw param.")
	}
	// No raw optional parameters after raw parameters.
	if err = cmd.AddRawParam("boo", "", false, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of optional raw param after optional raw param.")
	}
	// No raw optional parameters with value after raw parameters.
	if err = cmd.AddParam("boo", "", "", false, &boo); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param after raw param.")
	}
	// No raw required parameters after raw parameters.
	if err = cmd.AddParam("boo", "", "", true, nil); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param after raw param.")
	}
	// No raw required parameters with value after raw parameters.
	if err = cmd.AddParam("boo", "", "", true, &boo); err == nil {
		t.Fatal("Failed detecting invalid registration of required prefixed param after raw param.")
	}
	// No command registration on a command with raw params.
	if _, err = cmd.AddCommand("boo", "", nil); err == nil {
		t.Fatal("Failed detecting invalid registration of a command on a command with defined raw args.")
	}
	// Parse ok.
	if err = cl.Parse([]string{"foo", "bar", "baz"}); err != nil {
		t.Fatal(err)
	}
	// Required raw parameter missing.
	if err = cl.Parse([]string{"foo"}); err == nil {
		t.Fatal("Failed detecting missing required raw parameter.")
	}
	if err = cl.Parse([]string{"foo", "bar", "baz", "boo"}); err == nil {
		t.Fatal("Failed detecting extra arguments for registered raw params.")
	}
}

// Unregistered raw params.
func TestNoRegisteredParams(t *testing.T) {
	var handlererror = errors.New("propagated error")
	var fooArgs = []string{"foo"}
	var barArgs = []string{"bar", "one", "two", "three"}
	var bazArgs = []string{"baz", "one", "two", "three"}
	var foo = func(params Context) error {
		return handlererror
	}
	var bar = func(params Context) error {
		if len(params.Arguments()) != len(barArgs[1:]) {
			t.Fatal("Expected argument count does not match.")
		}
		for idx, arg := range params.Arguments() {
			if barArgs[idx+1] != arg {
				t.Fatal("Expected argument not found.")
			}
		}
		return nil
	}
	var cl = NewState()
	// Register a command returning an error from a handler.
	cl.MustAddRawCommand("foo", "", foo)
	// Register a command whose handler handles raw arguments.
	cl.MustAddRawCommand("bar", "", bar)
	// Register a command with no handler.
	cl.MustAddCommand("baz", "", nil)
	var err error
	// Propagate error from handler.
	if err = cl.Parse(fooArgs); err != handlererror {
		t.Fatal(err)
	}
	// Handler handles arguments.
	if err = cl.Parse(barArgs); err != nil {
		t.Fatal(err)
	}
	// Command receiving raw arguments must have a handler.
	if err = cl.Parse(bazArgs); err == nil {
		t.Fatal("Failed detecting command not having a handler for raw args.")
	}
}

// Test combined prefixed and raw parameters.
func TestCombined(t *testing.T) {
	var cl = NewState()
	var valBar, valBit string
	// Register "foo" command.
	cl.MustAddCommand("foo", "", nil).
		// Register "bar" prefixed param.
		MustAddParam("bar", "", "", true, &valBar).
		// Register "baz" prefixed optional param.
		MustAddParam("baz", "", "", false, nil).
		// Register a required raw param after prefixed params.
		MustAddRawParam("bit", "", true, &valBit).
		// Register an optional raw param after prefixed params.
		MustAddRawParam("bot", "", false, nil)
	var err error
	// Parse only required prefixed and raw parameters.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "bit"}); err != nil {
		t.Fatal(err)
	}
	// Parse all registered params.
	if err = cl.Parse([]string{"foo", "--bar", "bar", "--baz", "bit", "bot"}); err != nil {
		t.Fatal(err)
	}
	// Required prefixed parameter missing.
	if err = cl.Parse([]string{"foo", "bit"}); err == nil {
		t.Fatal("Failed detecting missing prefixed parameter.")
	}
	// Required raw parameter missing.
	if err = cl.Parse([]string{"foo", "--bar", "bar"}); err == nil {
		t.Fatal("Failed detecting missing required raw argument.")
	}

}

func BenchmarkRegisterCommand(b *testing.B) {
	var cl = NewState()
	var names = make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		names = append(names, fmt.Sprintf("command%d", i))
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cl.AddCommand(names[i], "", nil)
	}
}

func BenchmarkRegisterParam(b *testing.B) {
	var cl = NewState()
	var names = make([]string, 0, b.N)
	for i := 0; i < b.N; i++ {
		names = append(names, fmt.Sprintf("param%d", i))
	}
	var cmd = cl.MustAddCommand("foo", "", nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cmd.MustAddParam(names[i], "", "", false, nil)
	}
}

func BenchmarkParse(b *testing.B) {
	cl := NewState()
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

func ExampleState() {
	cl := NewState()
	var barVal string
	var cmdfunc = func(ctx Context) error {
		if ctx.Executed() && ctx.Name() == "baz" {
			fmt.Println("Hello from 'baz' Command.")
		}
		return nil
	}
	cl.MustAddCommand("", "", nil).MustAddParam("verbose", "v", "Verbose output.", false, nil)
	cl.MustAddCommand("foo", "Do the foo.", nil).MustAddParam("bar", "r", "Enable bar.", true, &barVal).
		MustAddCommand("baz", "Do the baz.", cmdfunc).MustAddRawParam("bat", "Enable bat.", false, nil)
	if err := cl.Parse([]string{"--verbose", "foo", "--bar", "bar", "baz", "bat"}); err != nil {
		panic(err)
	}
	fmt.Println(cl.Print())
	// Output: Hello from 'baz' Command.

	// [--verbose]     -v      Verbose output.

	// foo     Do the foo.
	// 		<--bar> -r      (string)        Enable bar.

	// 		baz     Do the baz.
	// 				[bat]   Enable bat.
}

func TestRepeat(t *testing.T) {
	var err error
	var state = NewState()
	var handler = func(ctx Context) error {
		var err error
		var value string
		var innerhandler = func(ctx Context) error {
			fmt.Println(ctx.Value("value"))
			return nil
		}
		var state = NewState()
		state.MustAddCommand("", "", innerhandler).MustAddParam("value", "v", "", false, &value)
		var args = ctx.Arguments()
		for len(args) > 0 {
			err = state.Parse(args)
			args = state.Arguments()
			if err != nil {
				if errors.Is(err, ErrDuplicateParameter) {
					if err = state.VisitMatches(); err != nil {
						return err
					}
					continue
				}
				fmt.Println(err)
				break
			}
		}
		return err
	}
	state.MustAddRawCommand("input", "", handler)
	if err = state.Parse([]string{"input", "--value", "one", "--value", "two"}); err != nil {
		t.Fatal(err)
	}
}

func TestNewPattern(t *testing.T) {
	var handler = func(ctx Context) error {
		fmt.Println(ctx.Value("test"))
		return nil
	}
	var commands = NewCommands(nil)
	commands.MustAddCommand("test", "", handler).
		MustAddParam("test", "t", "", false, nil)
	var err error
	if err = ParseArgs([]string{"test", "--test"}, commands); err != nil {
		t.Fatal(err)
	}
}
