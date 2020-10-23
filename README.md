# commandline

A command oriented command line parser.

Currently uses JSON by default to allow string to value conversion from arguments.

## Description

Commands can be defined in the central Parser type and handler functions of 
those commands are invoked when Command is specified on command line, which can
optionally abort the parse process. Command can have optional or required 
Param definitions which can write directly to Go values during parsing using 
JSON formatting and can also be analyzed in a Command handler function.
Commands can have Commands of their own allowing for a Command hierarchy.

Example:

```go
// Parsed Param can write directly to a Go value from command line arguments.
var verbose bool
var username string

// helptopic will be written during parsing of "help" command as we will define
// a Param that converts an argument to a Go value.
// It will be set to "sometopic" after parsing with the commandline shown 
// below the example.
var helptopic string

// cmdListUsers is a CommandFunc handler for "users" Command defined on "list" command.
func cmdListUsers(params *Params) error {

	// Reading a Go value modified by a Param during parsing.
	if verbose {
		fmt.Println("I'm way too verbose.")
	}

	// Accessing a parsed value via Params asserting its' type.
	username, ok := params.Long("username").Value().(string)
	if !ok {
		panic("No way this will happen.")
	}

	// Reading another modified Go value.
	log.Printf("User '%s' requested users list.\n", username)

	// Continue processing command line.
	return nil
}

// cmdHelp is a CommandRawFunc handler that can process raw arguments.
func cmdHelp(params []string) error {
	// Show help based on parsed Param value.
	fmt.Printf("User requested help on '%s'\n", helptopic)
}

// Create new Parser instance.
cl := New()

// Root Commands can have a single empty Command name to allow command line to
// start with Params rather than a Command which can be useful for allowing a
// "global params" pattern.
rootcmd, err := cl.AddCommand("", "Global flags.", nil)

// Register an optional "verbose" Params as global param.
rootcmd.AddParam("verbose", "v", "Be verbose.", false, &verbose)

// Register a "list" Command on the empty root Command to allow a Command to
// follow those "global params".
listcmd, err := rootcmd.AddCommand("list", "List various items.", nil)

// Register a "users" Command on the "list" Command. 
listcmd.AddCommand("users", "List users." cmdListUsers)

// Register a required Param for the "list" Command.
listcmd.AddParam("username", "u", "Specify username.", true, &username)

// Register a command with a raw arguments handler.
helpcmd, err := cl.AddCommand("help", "Show help, optionally for a command", cmdHelp)


// Register an optional raw parameter for "help" Command that specifies topic.
helpcmd.AddRawParam("command", "Specify command to get help for.", false, &helptopic)

// Parse the command line arguments.
// Returned error may be one of defined in commandline package denoting a Parse
// failure or an error returned by one of Command handler functions that aborts
// further parsing.
if err := cl.Parse(os.Args[1:]); err != nil {
	log.Fatal(err)
}
```

Valid command line for this example would be: '-v list users --username "foo"'

To execute the "help" Command: 'help sometopic'

## Status

Pretty clean, simple, fast and works. Nimble too.

What's left:
* Specialcasing for JSON values passed via command line. Especially compound types and quoting stuffs.
* Maybe abstract value parsing with codecs.
* Make a better printer with functional tab alignment and definitely move it from String().

## License

MIT

See included LICENSE file.