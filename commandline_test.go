package commandline

import (
	"testing"
)

func makeTest(t *testing.T) *Parser {

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

	return cl
}

func TestCommandLine(t *testing.T) {

	cl := makeTest(t)

	if err := cl.Parse([]string{"--verbose", "list", "-ac", "--username", "foo bar", "names"}); err != nil {
		t.Fatal(err)
	}
}

func BenchmarkCommandLine(b *testing.B) {
	b.StopTimer()
	cl := makeTest(nil)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		cl.Parse([]string{"--verbose", "list", "-ac", "--username", "foo", "names"})
	}
}
