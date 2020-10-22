package commandline

import (
	"testing"
)

func TestCommandLine(t *testing.T) {

	cmdGlobalFlags := func(params *Params) error {
		// fmt.Println("Verbose flag specified.")
		return nil
	}

	cl := New()
	cmd, err := cl.Register("", "global flags", cmdGlobalFlags)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params().Register("verbose", "v", "Enable verbose output.", false, nil); err != nil {
		t.Fatal(err)
	}

	cmd, err = cl.Register("list", "List items.", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params().Register("all", "a", "List all items.", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params().Register("color", "c", "Use color when listing.", false, nil); err != nil {
		t.Fatal(err)
	}
	username := ""
	if err := cmd.Params().Register("username", "u", "Specify username.", false, &username); err != nil {
		t.Fatal(err)
	}

	cmdListNames := func(params *Params) error {
		// fmt.Printf("User '%s' requested names list.\n", username)
		return nil
	}
	if _, err := cmd.Register("names", "List names.", cmdListNames); err != nil {
		t.Fatal(err)
	}

	if err := cl.Parse([]string{"--verbose", "list", "-ac", "--username", "foo bar", "names"}); err != nil {
		t.Fatal(err)
	}
}

func TestCustomHandler(t *testing.T) {

	testparams := []string{"one", "two", "three"}

	handler := func(args []string) error {
		for idx, val := range args {
			if testparams[idx] != val {
				t.Fatal("TestCustomHandler failed")
			}
		}
		return nil
	}

	cl := New()
	cmd, err := cl.Register("test", "", handler)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := cmd.Register("fail", "", nil); err == nil {
		t.Fatal("command allowed registering of sub-command on a command with a raw handler")
	}

	if err := cl.Parse(append([]string{"test"}, testparams...)); err != nil {
		t.Fatal(err)
	}
}
