# commandline

A lightweight, command-oriented command line parser.

## Description

Central Parser type registers Commands and invokes Handlers. Command can have optional 
or required Param definitions which can write directly to Go values and be
analyzed in a Handler.

Commands can have Commands of their own allowing for a Command hierarchy.

Params can be Prefixed (specified by name) or Raw (specified by index and 
addressable by name).

Commandline's only dependency outside of stdlib is 
[`github.com/vedranvuk/strconvex`](https://github.com/vedranvuk/strconvex)
which is a lightweight, reflect-based string to Go value converter used for 
converting command line arguments to their registered variables. Please
see that package for details on how Commandline converts strings to Go values.

## Examples

### Example with panic functions

```go
cl := New() // New Parser.
var barVal string // Receives value of 'bar' parameter.
// Command execution handler, in this example shared by multiple commands.
var cmdfunc = func(ctx Context) error {
	if ctx.Name() == "baz" && ctx.Arg("bat") != "bat" {
		return errors.New("I never asked for this.")
	}
	if ctx.Executed() && ctx.Name() == "baz" {
		fmt.Println("Hello from 'baz' Command.")
	}
	return nil
}
// Register a special 'global' command with an empty name used for preceeding 
// commands with params and register one prefixed param on it.
cl.MustAddCommand("", "", nil).MustAddParam("verbose", "v", "Verbose output.", false, nil)
// Register a 'foo' command with 'bar' prefixed param.
// Immediately register a 'baz' sub-command on 'foo' command with 'bat' raw paramater.
cl.MustAddCommand("foo", "Do the foo.", cmdfunc).MustAddParam("bar", "r", "Enable bar.", true, &barVal).
	MustAddCommand("baz", "Do the baz.", cmdfunc).MustAddRawParam("bat", "Enable bat.", false, nil)
// Parse parameters.
// Mark global empty command's verbose param as parsed.
// Invoke 'cmdfunc' for 'foo' command and set barVal to 'bar'.
// Invoke 'cmdfunc' for 'baz' command and set 'baz' parameter value to 'bat'.
if err := cl.Parse([]string{"--verbose", "foo", "--bar", "bar", "baz", "bat"}); err != nil {
	panic(err)
}
// Print the registered commands and their parameters.
fmt.Println(cl.Print())
// Output: Hello from 'baz' Command.

// [--verbose]     -v      Verbose output.

// foo     Do the foo.
// 		<--bar> -r      (string)        Enable bar.

// 		baz     Do the baz.
// 				[bat]   Enable bat.
```

### Example with error functions

```go
// Some variables that will receive values.
var (
	verbose     bool
	projectname string
	projectdir  string
)

// cmdGlobal gets invoked for "Global params" unnamed function.
func cmdGlobal(params Context) error {
	verbose = params.Parsed("verbose")
	return nil
}

// cmdCreate gets invoked for "Create project" command.
func cmdCreate(params Context) error {
	CreateProject(projectname, projectdir)
	return nil
}

// Create new parser. 
parser := New()

// Parser allows for a single unnamed command in root command definitions to
// allow the command line not to start with a command but parameters instead.
// If it is registered it must be the root command of all commands.
//
// Register a special "global flags" handler command that is automatically
// invoken if any of it's flags is specified by command line arguments and
// skipped if no "global flags" is given in arguments and the command line
// starts with a command.
cmd, err := parser.AddCommand("", "Global params", cmdGlobal)

// Register an optional "verbose" param on "global flags" command.
// It will not write to any Go value as pointer to one is nil.
cmd.AddParam("verbose", "v", "Be more verbose.", false, nil)

// Register a sub command for the "global flags" command so it can be invoken
// via arguments regardless if "global flags" executed.
cmd, err := cmd.AddCommand("create" "Create a project", cmdCreate)

// Add a prefixed parameter to "Create project" command that is required and
// converts an argument following the param to registered Go value projectname.
cmd.AddParam("name", "n", "Project name", true, &projectname)

// Add an optional raw param that isn't prefixed but is instead treated as a
// param value itself.
cmd.AddRawParam("directory", "Specify project directory", false, &projectdir)

// Parse command line, valid one would be "-v create --name myproject /home/me/myproject".
if err := parser.Parse(os.Args[1:]); err != nil {
	log.Fatal(err)
}
```

## Status

Working as intended. No API changes except additions planned.

What's left:
* ~~Maybe support complex structures as parameters.~~
* Maybe go:generate Parser definitions from Go composites.
* Maybe go:generate command handlers from a Parser instance.

## License

MIT

See included LICENSE file.