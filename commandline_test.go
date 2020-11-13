package commandline

import (
	"errors"
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
		t.Fatal("failed detecting non-existent command")
	}
}

func TestGlobal(t *testing.T) {
	cl := New()
	if err := cl.Parse([]string{"--verbose"}); err == nil {
		t.Fatal("failed detecting command not specified")
	}
	if err := cl.Parse([]string{"verbose"}); err == nil {
		t.Fatal("failed detecting unregistered command")
	}
	cl.MustAddCommand("", "", nil).MustAddParam("verbose", "v", "", false, nil)
	if err := cl.Parse([]string{"--verbose"}); err != nil {
		t.Fatal(err)
	}
	cl.MustAddCommand("foo", "", nil)
	if err := cl.Parse([]string{"--verbose", "foo"}); err != nil {
		t.Fatal(err)
	}
}

func TestNoRegisteredParams(t *testing.T) {
	ErrOK := errors.New("everything is fine")
	fooArgs := []string{"foo"}
	barArgs := []string{"bar", "one", "two", "three"}
	bazArgs := []string{"baz", "one", "two", "three"}
	cmdFoo := func(params *Params) error {
		return ErrOK
	}
	cmdBar := func(params *Params) error {
		for idx, arg := range params.RawArgs() {
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
	cmdFoo := func(params *Params) error {
		if params.RawValue("bar") != "bar" {
			t.Fatal("RegisteredRaw failed.")
		}
		if params.RawValue("baz") != "baz" {
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
	cmdFoo := func(params *Params) error {
		if params.RawValue("bar") != "bar" {
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
