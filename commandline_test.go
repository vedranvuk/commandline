package commandline

import (
	"fmt"
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
	if err := cmd.AddParam("all", "a", "List all items.", false, nil); err != nil {
		t.Fatal(err)
	}
	if err := cmd.AddParam("color", "c", "Use color when listing.", false, nil); err != nil {
		t.Fatal(err)
	}
	username := ""
	if err := cmd.AddParam("username", "u", "Specify username.", false, &username); err != nil {
		t.Fatal(err)
	}

	cmdListNames := func(Params *Params) error {
		// fmt.Printf("User '%s' requested names list.\n", username)
		return nil
	}
	if _, err := cmd.AddCommand("names", "List names.", cmdListNames); err != nil {
		t.Fatal(err)
	}

	fmt.Println(cl)

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

	fmt.Println(cl)

	if err := cl.Parse(append([]string{"test"}, testparams...)); err != nil {
		t.Fatal(err)
	}

	if one != "one" || two != "two" || three != "" {
		t.Fatal("Failed setting Param values.")
	}
}
