package commandline

import (
	"testing"
)

func TestHandler(t *testing.T) {

	cmdGlobalFlags := func(Params *Params) error {
		// fmt.Println("Verbose flag specified.")
		return nil
	}

	cmdDelete := func(Params []string) error {

		return nil
	}

	cl := New()
	cmd, err := cl.AddCommand("", "Global flags.", cmdGlobalFlags)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params.AddParam("verbose", "v", "Enable verbose output.", false, nil); err != nil {
		t.Fatal(err)
	}

	cmd, err = cl.AddCommand("delete", "Delete all files at specified path.", cmdDelete)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Params.AddParam("path", "p", "Path to target.", false, nil)
	cmd.Params.AddParam("recursive", "r", "Recursively delete all files in subfolders.", false, nil)

	cmd, err = cl.AddCommand("list", "List items.", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params.AddParam("all", "a", "List all items.", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params.AddParam("color", "c", "Use color when listing.", false, nil); err != nil {
		t.Fatal(err)
	}
	username := ""
	if err := cmd.Params.AddParam("username", "u", "Specify username.", false, &username); err != nil {
		t.Fatal(err)
	}

	cmdListNames := func(Params *Params) error {
		// fmt.Printf("User '%s' requested names list.\n", username)
		return nil
	}
	if _, err := cmd.AddCommand("names", "List names.", cmdListNames); err != nil {
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
	cmd, err := cl.AddCommand("test", "", handler)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := cmd.AddCommand("fail", "", nil); err == nil {
		t.Fatal("command allowed registering of sub-command on a command with a raw handler")
	}

	one := ""
	two := ""
	three := ""

	if err := cmd.Params.AddRawParam("One", "First parameter", 0, true, &one); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params.AddRawParam("Two", "Second parameter", 1, true, &two); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params.AddRawParam("Seventeen", "Seventeenth parameter", 17, false, nil); err == nil {
		t.Fatal("Allowed setting a raw param at out of order index.")
	}
	if err := cmd.Params.AddRawParam("Three", "Third parameter", 2, false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.Params.AddRawParam("Fourth", "Fourth parameter", 3, true, nil); err == nil {
		t.Fatal("Allowed registration of required parameter after non-required parameter")
	}

	if err := cl.Parse(append([]string{"test"}, testparams...)); err != nil {
		t.Fatal(err)
	}

	if one != "one" || two != "two" || three != "" {
		t.Fatal("Failed setting Param values.")
	}
}
