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

// Some variables that will receive values.
var (
	verbose     bool
	projectname string
	projectdir  string
)

// cmdGlobal gets invoked for "Global params" unnamed function.
func cmdGlobal(params *Params) error {
	verbose = params.Parsed("verbose")
	return nil
}

// cmdCreate gets invoked for "Create project" command.
func cmdCreate(params *Params) error {
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

No API changes except additions planned.

What's left:
* Specialcasing for JSON values passed via command line. Especially compound types and quoting stuffs.
* Maybe abstract value parsing with codecs.

## License

MIT

See included LICENSE file.