# commandline

A command oriented command line parser.

## Description

Central Parser type registers Commands with Handlers. Command can have optional 
or required Param definitions which can write directly to Go values and be
analyzed in a Handler.

Commands can have Commands of their own allowing for a Command hierarchy.

Params can be Prefixed (specified by name) or Raw (specified by index and 
addressable by name).

Example with panic functions:

```go
// New parser.
cl := New()
// To store value parsed for foo's bar param.
var barVal string
// Add a special unnamed command to root to hold 'global' params.
// Directly register a 'verbose' param on it. 
cl.MustAddCommand("", "", nil).MustAddParam("verbose", "v", "Verbose output.", false, nil)
// Add a 'foo' command and directly register a 'bar' prefixed param on it.
cl.MustAddCommand("foo", "Do the foo.", nil).MustAddParam("bar", "r", "Enable bar.", true, &barVal)
// Add a 'baz' command and directly register a 'bat' raw param on it.
cl.MustAddCommand("baz", "Do the baz.", nil).MustAddRawParam("bat", "Enable bat.", false, nil)
// Parse global verbose flag, execute 'foo' command and pass it '--bar' param 
// with value 'bar' that is written to &barVal, execute baz command and read 
// 'bat' as its' bat param value.
cl.Parse([]string{"--verbose", "foo", "--bar", "bar", "baz", "bat"})
fmt.Println(cl.Print())
// Output:
// --verbose       -v      Verbose output.

// foo     Do the foo.
//         --bar   -r      (string)        Enable bar.

// baz     Do the baz.
//         [bat]           Enable bat.
```

Example with error functions:

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

// Parse command line.
if err := parser.Parse(os.Args[1:]); err != nil {
	log.Fatal(err)
}
```

Valid command line for this example would be: `-v create --name myproject /home/me/myproject`

## Status

Working as intended. No API changes except additions planned.

What's left:
* Reflect incomming, will replace JSON with a light string converter.
* Maybe abstract value parsing with codecs.
* Maybe support complex structures as parameters.
* Maybe add a reflector from Go composites to preconstructed Parser.
* Maybe go:generate command handlers from a Parser instance.

## License

MIT

See included LICENSE file.