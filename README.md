# commandline

A command oriented command line parser.

Uses JSON to allow for complex command line arguments and value binding.

## Description

Commands can be defined in the central Parser type and handler functions of those commands are invoked when Command is specified on command line, which can optionally abort the parse process. Command can have optional or required Params. Params when specified on command line can write directly to Go values or be analyzed in a Command handler functions. Commands can have Commands of their own allowing for a Command hierarchy.

Example:

```go
// Parsed Param can write directly to a Go value from command line arguments.
var verbose bool
var username string

// cmdListUsers is a handler for "users" Command defined on "list" command.
func cmdListUsers(ps *ParamSet) error {

	// Reading a Go value modified by a Param during parsing.
	if verbose {
		fmt.Println("I'm way too verbose.")
	}

	// Accessing a parsed value via Params asserting its' type.
	username, ok := ps.Long("username").Value().(string)
	if !ok {
		panic("No way this will happen.")
	}

	// Reading another modified Go value.
	log.Printf("User '%s' requested users list.\n", username)

	// Continue processing command line.
	return nil
}

// Create new Parser instance.
cl := New()

// Root Commands can have a single empty Command name to allow command line to
// start with Params rather than a Command which can be useful for allowing a
// "global params" pattern.
rootcmd, err := cl.Register("", "Global flags.", nil)

// Register an optional "verbose" Params as global param.
rootcmd.Params().Register("verbose", "v", "Be verbose.", false, &verbose)

// Register a "list" Command on the empty root Command to allow a Command to
// follow those "global params".
listcmd, err := rootcmd.Register("list", "List various items.", nil)

// Register a "users" Command on the "list" Command. 
listcmd.Register("users", "List users." cmdListUsers)

// Register a required Param for the "list" Command.
listcmd.Params().Register("username", "u", "Specify username.", true, &username)

// Parse the command line arguments.
// Returned error may be one of defined in commandline package denoting a Parse
// failure or an error returned by one of Command handler functions that aborts
// further parsing.
if err := cl.Parse(os.Args[1:]); err != nil {
	log.Fatal(err)
}
```

Valid command line for this example would be: '-v list users --username "foo"'

## Status

Pretty clean, simple, fast and works.

As JSON is parsed from command line requires further specialcasing and a few more tests.

## License

MIT

See included LICENSE file.