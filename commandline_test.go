package commandline

import (
	"errors"
	"testing"
)

func TestHandler(t *testing.T) {

	cmdGlobalFlags := func(params *Params) error {
		return nil
	}

	cmdDelete := func(params *Params) error {
		return nil
	}

	cl := New()
	cmd, err := cl.AddCommand("", "Global flags.", cmdGlobalFlags)
	if err != nil {
		t.Fatal(err)
	}

	if c, ok := cl.GetCommand(""); !ok {
		t.Fatal("GetCommand failed to find a command.")
	} else {
		if c != cmd {
			t.Fatal("GetCommand returned the wrong Command.")
		}
	}

	if err := cmd.AddParam("verbose", "v", "Enable verbose output.", false, nil); err != nil {
		t.Fatal(err)
	}

	cmd, err = cl.AddCommand("delete", "Delete all files at specified path.", cmdDelete)
	if err != nil {
		t.Fatal(err)
	}
	cmd.AddParam("path", "p", "Path to target.", false, nil)
	cmd.AddParam("recursive", "r", "Recursively delete all files in subfolders.", false, nil)

	cmd, err = cl.AddCommand("list", "List items.", nil)
	if err != nil {
		t.Fatal(err)
	}

	if _, err = cl.AddCommand("list", "This duplicate must not register.", nil); err == nil {
		t.Fatal("Failed detecting adding a command with a duplicate name.")
	}

	if _, err := cmd.AddCommand("", "This must not register", nil); err != ErrInvalidName {
		t.Fatal("Failed detecting registering an empty command in a sub-Command.")
	}

	if err := cmd.AddParam("all", "a", "List all items.", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddParam("color", "c", "Use color when listing.", false, nil); err != nil {
		t.Fatal(err)
	}
	username := ""
	if err := cmd.AddParam("u", "username", "Specify username.", true, &username); err == nil {
		t.Fatal("Failed detecting param short name length.")
	}
	if err := cmd.AddParam("username", "u", "Specify username.", true, nil); err == nil {
		t.Fatal("Failed detecting required value for a required parameter.")
	}
	if err := cmd.AddParam("username", "u", "Specify username.", true, "nope"); err == nil {
		t.Fatal("Failed detecting required value type for a required parameter.")
	}
	if err := cmd.AddParam("username", "u", "Specify username.", true, &username); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddParam("username", "u", "Specify username.", true, &username); err == nil {
		t.Fatal("Failed detecting duplicate Param long name.")
	}
	if err := cmd.AddParam("username2", "u", "Specify username.", true, &username); err == nil {
		t.Fatal("Failed detecting duplicate Param short name.")
	}
	/*
		if err := cmd.AddRawParam("norawallowed", "Should not register on a Command with a CommandFunc handler.", false, nil); err == nil {
			t.Fatal("Failed detecting raw param registration on normal handler.")
		}
	*/

	cmdListNames := func(Params *Params) error {
		// fmt.Printf("User '%s' requested names list.\n", username)
		return nil
	}
	if _, err := cmd.AddCommand("names", "List names.", cmdListNames); err != nil {
		t.Fatal(err)
	}

	if err := cl.Parse([]string{}); err != ErrNoArgs {
		t.Fatal("Failed detecting no args.")
	}

	if err := cl.Parse([]string{""}); err != nil {
		t.Fatal(err)
	}

	if err := cl.Parse([]string{"list", "-acu", "foo bar"}); err == nil {
		t.Fatal("Failed detecting combined param with required value.")
	}

	if err := cl.Parse([]string{"list", "--username"}); err == nil {
		t.Fatal("Failed detecting required Param value.")
	}

	if err := cl.Parse([]string{"nosuchcommand"}); err == nil {
		t.Fatal("Failed detecting non-existent command.")
	}

	if err := cl.Parse([]string{"--nosuchparam"}); err == nil {
		t.Fatal("Failed detecting non-existent param.")
	}

	if err := cl.Parse([]string{"-v", "list", "-ac", "-u", "foo", "--whatnow"}); err == nil {
		t.Fatal("Failed detecting non-existent parameter.")
	}

	if err := cl.Parse([]string{"--verbose", "list", "-ac", "names"}); err == nil {
		t.Fatal("Failed detecting required Param.")
	}

	if err := cl.Parse([]string{"list", "-x"}); err == nil {
		t.Fatal("Failed detecting non-existent short parameter.")
	}

	if err := cl.Parse([]string{"list", "-acx"}); err == nil {
		t.Fatal("Failed detecting non-existent combined short parameter.")
	}

	if err := cl.Parse([]string{"--verbose", "list", "-ac", "--username", "foo bar", "names"}); err != nil {
		t.Fatal(err)
	}

	if err := cl.Parse([]string{"-v", "list", "--all", "-c", "-u", "foo bar", "names"}); err != nil {
		t.Fatal(err)
	}

}

func TestRegisteredRaw(t *testing.T) {

	testparams := []string{"one", "two", "three"}

	handler := func(params *Params) error {
		for idx, val := range params.RawArgs() {
			if testparams[idx] != val {
				t.Fatal("TestCustomHandler failed")
			}
		}
		return nil
	}

	cl := New()
	cmd, err := cl.AddCommand("test", "", handler)
	if err != nil {
		t.Fatal(err)
	}

	one := ""
	two := ""
	three := ""

	if err := cmd.AddRawParam("One", "First parameter", true, &one); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddRawParam("Two", "Second parameter", true, &two); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddRawParam("Three", "Third parameter", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddRawParam("Fourth", "Fourth parameter", true, nil); err == nil {
		t.Fatal("Allowed registration of required parameter after non-required parameter")
	}
	if err := cmd.AddRawParam("Fourth", "Fourth parameter", false, nil); err == nil {
		t.Fatal("Allowed registration of non-required parameter after non-required parameter")
	}

	if err := cl.Parse([]string{"--nonexistentparam"}); err == nil {
		t.Fatal("Failed detecting a non-global param.")
	}

	if err := cl.Parse(append([]string{"test"}, "one")); err == nil {
		t.Fatal("Failed detecting missing required argument.")
	}

	if err := cl.Parse(append(append([]string{"test"}, testparams...), "four")); err == nil {
		t.Fatal("Failed detecting extra arguments for registered params.")
	}

	if err := cl.Parse(append([]string{"test"}, testparams...)); err != nil {
		t.Fatal(err)
	}

	if one != "one" || two != "two" || three != "" {
		t.Fatal("Failed setting Param values.")
	}
}

func TestUnregisteredRaw(t *testing.T) {

	testArgs := []string{"test", "one", "two", "three"}

	ErrOK := errors.New("everything is fine")

	cmdTest := func(params *Params) error {
		for idx, arg := range params.RawArgs() {
			if testArgs[idx+1] != arg {
				t.Fatal("Unregistered param mode failed.")
			}
		}
		return ErrOK
	}

	cl := New()
	if _, err := cl.AddCommand("test", "Do da test.", cmdTest); err != nil {
		t.Fatal(err)
	}

	if err := cl.Parse(testArgs); err != ErrOK {
		t.Fatal("Failed propagating the error.")
	}
}

func TestCombinedParams(t *testing.T) {

	one := ""
	two := ""
	_ = two
	three := ""
	four := ""

	cmdTest := func(params *Params) error {
		/*
			fmt.Println(params.RawArgs())
			fmt.Println(one)
			fmt.Println(two)
			fmt.Println(three)
			fmt.Println(four)
		*/
		return nil
	}

	cl := New()

	cmd, err := cl.AddCommand("test", "Test command with mixed params.", cmdTest)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddParam("one", "", "First, required parameter", true, &one); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddParam("two", "", "Second, optional parameter", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddRawParam("three", "Third, required raw parameter", true, &three); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddRawParam("four", "Fourth, optional raw parameter", false, &four); err != nil {
		t.Fatal(err)
	}

	if err := cl.Parse([]string{"test", "--one", "1", "--two", "three", "four"}); err != nil {
		t.Fatal(err)
	}

	if err := cl.Parse([]string{"test", "--one", "1", "three", "four"}); err != nil {
		t.Fatal(err)
	}

}

func TestParsed(t *testing.T) {

	cmdTest := func(params *Params) error {
		if !params.Parsed("one") {
			t.Fatal("Parsed() failed: prefixed param not marked as parsed.")
		}
		if !params.Parsed("two") {
			t.Fatal("Parsed() failed: raw param not marked as parsed.")
		}
		return nil
	}

	cl := New()
	cmd, err := cl.AddCommand("parse", "Parse test", cmdTest)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddParam("one", "1", "Param one", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddRawParam("two", "Param two", false, nil); err != nil {
		t.Fatal(err)
	}

}
