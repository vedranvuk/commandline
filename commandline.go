// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package commandline implements a command line parser.
package commandline

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/vedranvuk/strconvex"
)

// ParseArgs parses arguments into commands and returns any errors.
func ParseArgs(arguments []string, commands *Commands) error {
	var state = &State{
		Commands: commands,
	}
	return state.Parse(arguments)
}

// ParseOsArgs parses os arguments into commands and returns any errors.
func ParseOsArgs(commands *Commands) error {
	return ParseArgs(os.Args[1:], commands)
}

// TODO Better printing. Use two spaces instead of tabs. Wrap to 80 chars.
// TODO Add Print() option to print commands in order of registration or name.

var (
	// ErrCmdline is the base error or commandline package.
	ErrCommandline = errors.New("commandline")

	// ErrConvert or a descendant of it is returned when an error occurs on
	// converting an argument to a go value.
	ErrConvert = fmt.Errorf("%w: convert", ErrCommandline)

	// ErrRegister is the base Command or Parameter registration error.
	ErrRegister = fmt.Errorf("%w: register", ErrCommandline)
	// ErrDuplicate help
	ErrDuplicate = fmt.Errorf("%w: duplicate", ErrRegister)

	// ErrParse is the base parse error.
	ErrParse = fmt.Errorf("%w: parse error", ErrCommandline)
	// ErrInvalidArgument is returned when an invalid argument was encountered.
	ErrInvalidArgument = fmt.Errorf("%w: invalid argument", ErrParse)
	// ErrNoArguments is returned by Commands and Parameters Parse methods if
	// no arguments were left to parse from State.
	ErrNoArguments = fmt.Errorf("%w: no arguments", ErrParse)
	// ErrNoDefinitions is returned by Commands and Parameters Parse methods if
	// no Commands or Parameters were defined to parse.
	ErrNoDefinitions = fmt.Errorf("%w: no definitions", ErrParse)
	// ErrNotFound is returned when a Command or Parameter is not found.
	ErrNotFound = fmt.Errorf("%w: not found", ErrParse)
	// ErrDuplicateParameter is returned when a duplicate parameter was parsed.
	ErrDuplicateParameter = fmt.Errorf("%w: parameter repeats", ErrParse)
	// ErrExtraArguments is returned when extra arguments are specified and
	// last commands is not a raw argument handler.
	ErrExtraArguments = fmt.Errorf("%w: extra arguments", ErrParse)
)

// Context is a CommandFunc context that provides info about Command execution.
type Context interface {
	// Arguments returns unparsed command line arguments if command is raw.
	Arguments() []string
	// Name returns the name of Command which registered this handler.
	Name() string
	// Print prints the calling Command definition and any of its Commands as
	// structured text suitable for terminal display.
	Print() string
	// Value returns the raw string argument given to a parameter under 
	// specified name. If parameter was not parsed or is not registered 
	// an empty string is returned.
	Value(string) string
	// Executed will be true if context is from a handler whose command is the
	// last command in the chain matched from command line.
	Executed() bool
	// Parsed returns true if the parameter under specified long name is defined
	// and parsed from command line and false otherwise.
	Parsed(string) bool
}

// Handler is a prototype of a function that handles the event of a
// Command being parsed from command line arguments. A command handler.
//
// CommandFuncs of all Commands in the chain parsed on command line are
// visited during Parser.Parse(). Only the last matched command is marked
// as executed and can be discerned from visited CommandFuncs using
// Context.Executed().
//
// If a Handler returns a non-nil error further calling of handlers of
// Commands parsed from command line arguments is aborted and the error is
// propagated to Parse method and returned.
type Handler = func(Context) error

// context is the Context adapter. It wraps a Command and returns its'
// properties and the properties of its' Parameters in a single type that directly
// implements Context interface.
type context struct {
	executed  bool
	cmd       *Command
	arguments []string
}

// Name implements Context.Name.
func (c *context) Name() string { return c.cmd.name }

// Executed implements Context.Executed.
func (c *context) Executed() bool { return c.executed }

// Parsed implements Context.Parsed.
func (c *context) Parsed(name string) bool {
	var param *Parameter
	var exists bool
	if param, exists = c.cmd.Parameters.longparams[name]; exists {
		return param.parsed
	}
	return false
}

// Arg implements Context.Arg.
func (c *context) Value(name string) string {
	var param *Parameter
	var exists bool
	if param, exists = c.cmd.Parameters.longparams[name]; exists {
		return param.rawvalue
	}
	return ""
}

// Args implements Context.Args.
func (c *context) Arguments() []string { return c.arguments }

// Print implements Context.Print.
func (c *context) Print() string { return c.cmd.Print() }

// exec executes the context's command and returns its' handler return value.
func (c *context) exec() error {
	if c.cmd.handler == nil {
		return nil
	}
	return c.cmd.handler(c)
}

// Argument defines type of argument as recognized from command line.
type Argument int

const (
	// InvalidArgument represents an invalid argument.
	InvalidArgument Argument = iota
	// NoArgument represents no argument.
	NoArgument
	// TestArgument represents a single, possibly text separated text argument.
	TextArgument
	// LongArgument represents a word argument directly preffixed with "--".
	LongArgument
	// ShortArgument represents a char argument directly preffixed with "-".
	ShortArgument
	// CombinedArgument represents a word argument directly preffixed with "-".
	CombinedArgument
)

// String implements stringer on argKind.
func (ak Argument) String() (s string) {
	switch ak {
	case InvalidArgument:
		s = "invalid"
	case NoArgument:
		s = "none"
	case TextArgument:
		s = "command or raw argument"
	case LongArgument:
		s = "long parameter"
	case ShortArgument:
		s = "short parameter"
	case CombinedArgument:
		s = "combined short arguments"
	}
	return
}

// State is a command line parser. Its' Parse method is to be invoked
// with a slice of command line arguments passed to program.
// For example:
//  err := State.Parse(os.Args[1:])
//
// It is command oriented, meaning that one or more Command instances can be
// defined in State's Commands which when parsed from command line arguments
// invoke CommandFuncs registered for those Command instances. Command can have
// its' own Commands so a Command hierarchy can be defined.
//
// Root Commands, as an exception, allow for one Command with an empty name
// to be defined. This is to allow that program args need not start with a
// Command and to allow Parameters to be passed first which can act as "global".
// e.g. "--verbose list users".
//
// Command can have one or more Param instances defined in its' Parameters which
// can have names, help text, be required or optional and have an optional
// pointer to a Go value which is written from a value following Param in
// command line arguments.
//
// If a pointer to a Go value is registered with a Param, the Param will require
// an argument following it that the parser will try to convert to the Go value
// registered with Param. Otherwise the Param will act as a simple flag which
// can be checked if parsed in the Command handler by checking the result of
// handler's Context.Parsed() method.
//
// State supports prefixed and raw params which can be combined on a Command
// with a caveat that the Command that has one or more raw params registered
// cannot have sub-Commands because of ambiguities in parsing command names and
// raw parameters as well as the fact that one last raw param can be optional.
//
// Prefixed params are explicitly addressed on a command line and can have
// short and long forms. They can be marked optional or required and be
// registered in any order, but must be defined before any raw params.
//
// Short Param names have the "-" prefix, can be one character long and can be
// combined together following the short form prefix if none of the combined
// Parameters require a Param Value. All combined short params must all be optional.
//
// Long Param names have the "--" prefix and cannot be combined.
//
// Raw params are not addressed but are instead matched against registered raw
// Parameters in order of registration as they appear on command line, respectively.
//
// Prefixed and raw params can both be registered for a Command but raw params
// must be registered last and specified after prefixed Parameters on the command
// line.
// e.g. "rmdir -v /home/me/stuff" where "rmdir" is a command, "-v" is a
// prefixed param and "/home/me/stuff" is a raw parameter.
//
// If Parameters are defined as optional they do not cause a parse error if not
// parsed from program args and return a parse error if defined as required and
// not parsed from command line.
//
// For specifics on how Parameters are parsed see AddParam and AddRawParam help.
//
// Commands can have a CommandFunc registered optionaly so that a Command can
// serve solely as sub-Command selector. For more details see CommandFunc.
//
// If no Parameters were defined on a Command all command line arguments following
// the command invocation will be passed to Command handler via Parameters.Args.
//
// If no params were defined on a Command and the command has no CommandFunc
// registered an error is returned.
type State struct {
	// arguments is a slice of arguments being parsed.
	// Args are set once by Parse() then read and updated by Commands
	// and Parameters down the Parse chain until exhausted or an error occurs.
	arguments []string
	// matches is a slice of commands parsed from command line in the
	// order as they were parsed.
	matches []*Command
	// Commands is the root command set.
	*Commands
}

// NewState returns a new State instance initialized to specified arguments.
func NewState() *State {
	var state = &State{}
	state.Commands = NewCommands(nil)
	return state
}

// Extra returns current arguments in State.
func (p *State) Arguments() []string { return p.arguments }

// ArgumentCount returns current number of arguments.
func (state *State) ArgumentCount() int { return len(state.arguments) }

// Print prints the Parser as currently configured.
// Returns output suitable for terminal display.
func (state State) Print() string {
	sb := &strings.Builder{}
	printCommands(sb, state.Commands, 0)
	return sb.String()
}

// Parse parses specified args, usually invoked as "Parse(os.Args[1:])".
// If a parse error occurs or an invoked Command handler returns an error
// it is returned. Returns ErrNoArgs if args are empty and there are defined
// Commands or Parameters.
//
// TODO Remove.
func (state *State) Parse(args []string) error {
	state.reset()
	state.arguments = args
	var err = state.Commands.Parse(state)
	if err == nil {
		return state.VisitMatches()
	}
	// There are unparsed arguments.
	if errors.Is(err, ErrNotFound) {
		// There were no matches, error is due to unregistered command.
		if len(state.matches) == 0 {
			return err
		}
		// If last matched command is not a raw handler...
		if !state.lastMatch().Raw() {
			// No handler at all,
			if state.lastMatch().Handler() == nil {
				return errors.New("no handler for arguments.")
			}
			// ...arguments are extra.
			return ErrExtraArguments
		}
		return state.VisitMatches()
	}
	// Pass through as top error only if nothing was matched. If it was,
	// it is a result due to all arguments being consumed and successfull
	// parse by Commands or Parameters down the chain.
	if len(state.matches) == 0 {
		return err
	}
	return err
}

// Peek returns the first arg in args if args are not empty, otherwise returns
// an empty string.
func (p *State) Peek() string {
	if len(p.arguments) > 0 {
		return p.arguments[0]
	}
	return ""
}

// Next returns the first argument in Parser arguments trimmed of any prefixes
// and its' kind. If the argument is malformed returns an empty arg
// kind == argInvalid.
func (p *State) Next() (arg string, kind Argument) {
	kind = NoArgument
	if len(p.arguments) == 0 {
		return
	}
	arg = p.arguments[0]
	if len(arg) == 0 {
		return
	}
	for i, l := 0, len(arg); i < l; i++ {
		if arg[0] != '-' {
			break
		}
		arg = arg[1:]
		if kind == NoArgument {
			kind = ShortArgument
			continue
		}
		if kind == ShortArgument {
			kind = LongArgument
			continue
		}
		if kind == LongArgument {
			return "", InvalidArgument
		}
	}
	if len(arg) == 0 {
		return "", InvalidArgument
	}
	if kind == NoArgument {
		kind = TextArgument
		return
	}
	if kind == ShortArgument && len(arg) > 1 {
		return arg, CombinedArgument
	}
	return
}

// Skip discards the first arg in the args slice and returns a bool indicating
// if there is any args left.
func (p *State) Skip() bool {
	if len(p.arguments) == 0 {
		return false
	}
	p.arguments = p.arguments[1:]
	return len(p.arguments) > 0
}

// AddMatch adds command to a list of matches.
func (p *State) AddMatch(command *Command) {
	p.matches = append(p.matches, command)
}

// VisitMatches visits all matched commands, constructs a context and calls
// their handlers. Propagates first non-nil return value of visited handler.
func (p *State) VisitMatches() error {
	var l = len(p.matches)
	if l < 1 {
		return nil
	}
	var ctx = context{
		arguments: p.arguments,
	}
	var i int
	var err error
	for i = 0; i < l-1; i++ {
		ctx.cmd = p.matches[i]
		ctx.executed = false
		if err = ctx.exec(); err != nil {
			return err
		}
	}
	ctx.cmd = p.matches[i]
	ctx.executed = true
	return ctx.exec()
}

// lastMatch help.
func (p *State) lastMatch() *Command {
	if len(p.matches) == 0 {
		return nil
	}
	return p.matches[len(p.matches)-1]
}

// reset resets states of Parser, Commands and their params.
func (state *State) reset() {
	state.matches = []*Command{}
	resetCommands(state.Commands)
}

// Command is a command definition.
type Command struct {
	help        string  // help is the help text.
	handler     Handler // handler is the command handler. Can be nil.
	raw         bool
	*Parameters // Parameters are this Command's Parameters.
	*Commands   // Commands are this Command's sub Commands.
}

// NewCommand returns a new Command instance with specified optional help and
// handler optional if raw is false. If raw is true command's handler will
// receive unparsed parameters for custom handling.
func NewCommand(help string, handler Handler, raw bool) *Command {
	p := &Command{
		help:    help,
		handler: handler,
		raw:     raw,
	}
	p.Parameters = newParameters(p)
	p.Commands = NewCommands(p)
	return p
}

// Help help.
func (c *Command) Help() string { return c.help }


// Handler help.
func (c *Command) Handler() Handler { return c.handler }

// Raw help.
func (c *Command) Raw() bool { return c.raw }

// nameToCommand is a map of command name to *Command.
type nameToCommand map[string]*Command

// Commands holds a set of Commands with a unique name.
type Commands struct {
	// parent is this Command's parent.
	// If it is a Parser these Commands are the root Commands.
	// If it is a Command these are Command's sub commands.
	parent *Command
	// name is the Command name.
	name string
	// commandmap is a map of command names to *Command definitions.
	commandmap nameToCommand
	// nameindexes is a slice of command names in order as they were defined.
	nameindexes []string
}

// NewCommands returns a new Commands instance with specified parent which can
// be nil.
func NewCommands(parent *Command) *Commands {
	return &Commands{
		parent:     parent,
		commandmap: make(nameToCommand),
	}
}

// CommandCount returns number of registered commands.
func (c *Commands) CommandCount() int { return len(c.commandmap) }

// Print prints Commands as a structured text suitable for terminal display.
func (c *Commands) Print() string {
	var sb = &strings.Builder{}
	printCommands(sb, c, 0)
	return sb.String()
}

// AddCommand registers a new Command under specified name and help text that
// invokes handler when parsed from arguments.
//
// Command with an empty name allows for passing just parameters to Commands set
// when parsing and is executed in parallel with another named command in
// same Commands set.
//
// Order of registration is important. When printed, Commands are listed in the
// order they were registered instead of name sorted.
//
// If an error occurs Command will be nil and error will be ErrRegister or a
// descendant of it.
func (c *Commands) AddCommand(name, help string, handler Handler) (*Command, error) {
	return c.addCommand(name, help, handler, false)
}

// MustAddCommand is like AddCommand except the function panics on error.
// Returns added *Command.
func (c *Commands) MustAddCommand(name, help string, f Handler) *Command {
	var cmd, err = c.AddCommand(name, help, f)
	if err != nil {
		panic(err)
	}
	return cmd
}

// AddRawCommand is a command which when invoked last in the chain and there are
// unparsed arguments left does not cause an extra arguments error as it is
// marked as being able to handle, raw unparsed arguments. Raw command must have
// a valid handler or an error is returned.
func (c *Commands) AddRawCommand(name, help string, handler Handler) (*Command, error) {
	if handler == nil {
		return nil, errors.New("raw command must have a handler")
	}
	return c.addCommand(name, help, handler, true)
}

// MustAddRawCommand is like AddRawCommand but panics on error.
func (c *Commands) MustAddRawCommand(name, help string, handler Handler) *Command {
	var cmd, err = c.AddRawCommand(name, help, handler)
	if err != nil {
		panic(err)
	}
	return cmd
}

// GetCommand returns a *Command by name if found and truth if found.
func (c *Commands) GetCommand(name string) (cmd *Command, ok bool) {
	cmd, ok = c.commandmap[name]
	return
}

// MustGetCommand is like GetCommand but panics if Command is not found.
func (c *Commands) MustGetCommand(name string) *Command {
	var cmd *Command
	var ok bool
	if cmd, ok = c.commandmap[name]; ok {
		return cmd
	}
	panic(fmt.Sprintf("commandline: command '%s' not found", name))
}

// Parse parses Parser args into this Commands.
func (c *Commands) Parse(state *State) error {
	var err error
	var arg string
	var kind Argument
	var cmd *Command
	var ok, global bool
	switch arg, kind = state.Next(); kind {
	case InvalidArgument:
		return ErrInvalidArgument
	case NoArgument:
		return ErrNoArguments
	case TextArgument:
		if cmd, ok = c.commandmap[arg]; !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, arg)
		}
	default:
		if cmd, ok = c.commandmap[""]; !ok {
			return fmt.Errorf("%w: %s", ErrNotFound, arg)
		}
		global = true
	}
	// A command was matched at this point.
	// If an empty command is registered and argument was a param
	// threat it as a param to that command. Don't skip it so it can
	// be parsed py Paremeters.
	if !global {
		state.Skip()
	}
	// Parse Parameters.
	// If all required parameters parsed returns nil.
	// If no registered parameters returns ErrNoDefinitions.
	// If no arguments left in state returns ErrNoArguments.
	// If parameter not found returns ErrNotFound.
	// If paremeter repeats returns ErrDuplicateParameter.
	// Returns other parse specific errors.
	if err = cmd.Parameters.Parse(state); err != nil {
		if !errors.Is(err, ErrNoArguments) && !errors.Is(err, ErrNoDefinitions) {
			return err
		}
	}
	// Append command to matched commands.
	state.AddMatch(cmd)
	// Repeat parse on these Commands so that next command
	// invocation in arguments is passed back to these Commands.
	// Empty command is executed in parallel to other commands.
	if global {
		err = c.Parse(state)
	} else {
		err = cmd.Commands.Parse(state)
	}
	// Pass control to contained Commands to continue chaining.
	// They will return ErrNotFound or a descendant if
	// No commands or parameters were matched.
	if err != nil {
		if errors.Is(err, ErrNoArguments) {
			return nil
		}
	}
	return err
}

// addCommand registers a new command under specified name and help with
// specified handler and marks it raw if raw if true. If an error occurs it will
// be ErrRegister or a descendant and command will be nil.
//
// Name specifies command name which is unique in Commands. An empty name
// registers an empty command which is executed in parallel to other registered
// commands, i.e. it allows for omitting the command name when parsing Commands
// and instead just passes control to parsing empty command's parameters.
//
// Help is optional and can be anything. Handler is optional if raw is not true.
// If Raw is true Command will accept any unparsed arguments for custom hadling.
func (c *Commands) addCommand(name, help string, handler Handler, raw bool) (*Command, error) {
	var ok bool
	// No duplicate names.
	if _, ok = c.commandmap[name]; ok {
		if name == "" {
			return nil, fmt.Errorf("%w: empty command", ErrDuplicate)
		}
		return nil, fmt.Errorf("%w: command: '%s'", ErrDuplicate, name)
	}
	// Ambiguity of sub command name with parent with optional raw arguments.
	if c.parent != nil {
		if c.parent.HasOptionalRawArgs() {
			return nil, fmt.Errorf("%w: command with raw parameters cannot have sub commands", ErrRegister)
		}
	}
	// Define and add a new Command to self.
	var cmd = NewCommand(help, handler, raw)
	cmd.name = name
	c.commandmap[name] = cmd
	c.nameindexes = append(c.nameindexes, name)
	return cmd, nil
}

// Parameter defines a Command parameter contained in a Parameters.
type Parameter struct {
	// help is the Param help text.
	help string
	// rawvalue is the raw parsed param value, possibly empty.
	rawvalue string
	// value is a pointer to a Go value which is set
	// from parsed Param value if not nil and points to a
	// valid target.
	value interface{}
	// raw specifies if this param is a raw param.
	raw bool
	// required specifies if this Param is required.
	required bool
	// parsed indicates if Param was parsed from arguments.
	parsed bool
}

// NewParameter returns a new *Param instance with given help, required and value.
func NewParameter(help string, required, raw bool, value interface{}) *Parameter {
	return &Parameter{
		help:     help,
		required: required,
		raw:      raw,
		value:    value,
	}
}

// nameToParameter maps a param name to *Param.
type nameToParameter map[string]*Parameter

// longToShort maps a long param name to short param name.
type longToShort map[string]string

// A Parameters defines a set of Command Parameters unique by long name.
type Parameters struct {
	// cmd is the reference to owner *Command.
	cmd *Command
	// longparams is a map of long param name to *Param.
	longparams nameToParameter
	// shortparams is a map of short param name to *Param.
	shortparams nameToParameter
	// longtoshort maps a long param name to short param name.
	longtoshort longToShort
	// longindexes hold long param names in order as they are added.
	longindexes []string
}

// newParameters returns a new instance of *Parameters.
func newParameters(cmd *Command) *Parameters {
	return &Parameters{
		cmd,
		make(nameToParameter),
		make(nameToParameter),
		make(longToShort),
		[]string{},
	}
}

// ParameterCount returns number of defined parameters.
func (p *Parameters) ParameterCount() int { return len(p.longindexes) }

// AddParam registers a new prefixed Param in these Parameters.
//
// Long param name is required, short is optional and can be empty, as is help.
//
// If required is specified value must be a pointer to a supported Go value
// which will be updated to a value parsed from an argument following param.
// If a required Param or its' value is not found in command line args an error
// is returned.
//
// If Param is not marked as required, specifying a value parameter is optional
// but dictates that:
// If nil, a value for the Param will not be parsed from args.
// If valid, the parser will parse the argument following the Param into it.
//
// Registration order is important. Prefixed params must be registered before
// raw params and are printed in order of registration.
//
// If an error occurs Param is not registered.
func (p *Parameters) AddParam(long, short, help string, required bool, value interface{}) error {
	return p.addParam(long, short, help, required, false, value)
}

// MustAddParam is like AddParam except the function panics on error.
// Returns a Command that the param was added to.
func (p *Parameters) MustAddParam(long, short, help string, required bool, value interface{}) *Command {
	var err error
	if err = p.AddParam(long, short, help, required, value); err != nil {
		panic(err)
	}
	return p.cmd
}

// AddRawParam registers a raw Param under specified name which must be unique
// in long Parameters names. Raw params can only be defined after prefixed
// params or other raw params. Calls to AddParam after AddRawParam will error.
//
// Registering a raw parameter for a command will disable command's ability to
// have sub commands registered as its invocation would be ambiguous with raw
// parameters during Parameters parsing. If the command already has sub commands
// registered the function will error.
//
// Parsed arguments are applied to registered raw Parameters in order as they are
// defined. If value is a pointer to a valid Go value argument will be converted
// to that Go value. Specifying a value is optional and if nil, parsed argument
// will not be parsed into the value.
//
// Marking a raw param as required does not imply that value must not be nil
// as is in prefixed params. Required flag solely returns a parse error if
// required raw param was not parsed and value is set only if non-nil.
//
// A single non-required raw Param is allowed and it must be the last one.
//
// If an error occurs it is returned and the Param is not registered.
func (p *Parameters) AddRawParam(name, help string, required bool, value interface{}) error {
	return p.addParam(name, "", help, required, true, value)
}

// MustAddRawParam is like AddRawParam except the function panics on error.
// Returns a Command that the param was added to.
func (p *Parameters) MustAddRawParam(name, help string, required bool, value interface{}) *Command {
	var err error
	if err = p.AddRawParam(name, help, required, value); err != nil {
		panic(err)
	}
	return p.cmd
}

// Parse parses self from state arguments and updates state.
// If an error occurs it will be an ErrParse or a descendant.
// Returns nil if all required parameters were parsed.
// Returns ErrNoDefinitions if no parameters are defined.
func (p *Parameters) Parse(state *State) error {
	var paramcount int = p.ParameterCount()
	if paramcount == 0 {
		return ErrNoDefinitions
	}
	var err error
	var arg string
	var kind Argument
	var param *Parameter
	var exists bool
	var i int
	for i = 0; i < paramcount; {
		arg, kind = state.Next()
		switch kind {
		case InvalidArgument:
			return ErrInvalidArgument
		case NoArgument:
			goto checkRequired
		case TextArgument:
			// Start of raw params, skip prefixed.
			for i < paramcount {
				if param = p.longparams[p.longindexes[i]]; !param.raw {
					i++
					continue
				}
				break
			}
			// No defined raw params, assume sub command name.
			if i >= paramcount {
				goto checkRequired
			}
			i++
		case ShortArgument:
			if param, exists = p.shortparams[arg]; !exists {
				return fmt.Errorf("%w: short parameter '%s'", ErrNotFound, arg)
			}
			i++
		case LongArgument:
			if param, exists = p.longparams[arg]; !exists {
				return fmt.Errorf("%w: long parameter '%s'", ErrNotFound, arg)
			}
			i++
		case CombinedArgument:
			// Parse all combined args and continue.
			var shorts = strings.Split(arg, "")
			var short string
			for _, short = range shorts {
				if param, exists = p.shortparams[short]; !exists {
					return fmt.Errorf("%w: short parameter '%s'", ErrNotFound, short)
				}
				if param.value != nil {
					return fmt.Errorf("%w: short parameter '%s' requires argument, cannot combine", ErrParse, short)
				}
				// Param is specified multiple times.
				if param.parsed {
					return fmt.Errorf("%w: combined parameter '%s' specified multiple times", ErrParse, short)
				}
				param.parsed = true
				i++
			}
			state.Skip()
			continue
		}
		// Param is specified multiple times.
		if param.parsed {
			return fmt.Errorf("%w: %s", ErrDuplicateParameter, arg)
		}
		// Parse value argument for params with value.
		if param.value != nil {
			// Advance argument for prefixed params.
			if !param.raw {
				if !state.Skip() {
					return fmt.Errorf("%w: parameter '%s' requires a value", ErrParse, arg)
				}
				arg = state.Peek()
			}
			// Set value.
			if err = stringToGoValue(arg, param.value); err != nil {
				return err
			}
		}
		// Advance.
		param.rawvalue = arg
		param.parsed = true
		if !state.Skip() {
			break
		}
	}
checkRequired:
	// Check all required params were parsed.
	for arg, param = range p.longparams {
		if param.required && !param.parsed {
			return fmt.Errorf("%w: required parameter '%s' not specified", ErrParse, arg)
		}
	}
	if state.ArgumentCount() == 0 {
		return ErrNoArguments
	}
	return nil
}

// addParam is the implementation of AddParam minus the checks of exposed API.
func (p *Parameters) addParam(long, short, help string, required, raw bool, value interface{}) error {
	// Long name must not be empty and short name must be max one char long.
	if long == "" || len(short) > 1 {
		return fmt.Errorf("%w: invalid name", ErrRegister)
	}
	// No long duplicates.
	var ok bool
	if _, ok = p.longparams[long]; ok {
		return fmt.Errorf("%w: long parameter name '%s'", ErrDuplicate, long)
	}
	// No short duplicates if not empty.
	if _, ok = p.shortparams[short]; ok && short != "" {
		return fmt.Errorf("%w: short parameter name '%s'", ErrDuplicate, short)
	}
	// Disallow adding optional raw parameters if command expects sub commands.
	if p.cmd.CommandCount() > 0 && raw && !required {
		return fmt.Errorf("%w: cannot register optional raw parameter on a command with sub commands", ErrRegister)
	}
	// Raw params can only be registered after prefixed params.
	// Optional raw params can only be registered after any required raw params.
	var param *Parameter
	if param = p.last(); param != nil && param.raw {
		if !raw {
			return fmt.Errorf("%w: cannot register prefixed parameter after raw parameter", ErrRegister)
		}
		if !param.required {
			if !required {
				return fmt.Errorf("%w: cannot register multiple optional parameters", ErrRegister)
			}
			return fmt.Errorf("%w: cannot register required after optional parameter", ErrRegister)
		}
	}
	// Required prefixed params need a valid Go value.
	if value == nil {
		if !raw && required {
			return fmt.Errorf("%w: value required", ErrRegister)
		}
	} else {
		// Value must be a valid pointer to a Go value.
		if v := reflect.ValueOf(value); !v.IsValid() || v.Kind() != reflect.Ptr {
			return fmt.Errorf("%w: invalid value", ErrRegister)
		}
	}
	// Register a new param.
	param = NewParameter(help, required, raw, value)
	p.longparams[long] = param
	if short != "" {
		p.shortparams[short] = param
	}
	p.longtoshort[long] = short
	p.longindexes = append(p.longindexes, long)
	return nil
}

// last returns the last defined arg or nil if none registered.
func (p *Parameters) last() *Parameter {
	if len(p.longindexes) == 0 {
		return nil
	}
	return p.longparams[p.longindexes[len(p.longindexes)-1]]
}

// HasOptionalRawArgs returns if Parameters contain one or more defined raw Parameters.
func (p *Parameters) HasOptionalRawArgs() bool {
	var param *Parameter
	for _, param = range p.longparams {
		if param.raw && !param.required {
			return true
		}
	}
	return false
}

// stringToGoValue converts a string to a Go value or returns an error.
func stringToGoValue(s string, i interface{}) error {
	if err := strconvex.StringToInterface(s, i); err != nil {
		return fmt.Errorf("%w: error converting value %s: %v", ErrConvert, s, err)
	}
	return nil
}

// resetCommands recursively resets all Commands and their Parameters states.
func resetCommands(c *Commands) {
	var cmd *Command
	var param *Parameter
	for _, cmd = range c.commandmap {
		if len(cmd.Parameters.longparams) > 0 {
			for _, param = range cmd.Parameters.longparams {
				param.parsed = false
				param.rawvalue = ""
			}
		}
		resetCommands(cmd.Commands)
	}
}

// printCommands is a recursive printer or registered Commands and Parameters.
// Lines are written to sb from current commands with the indent depth(*tab).
func printCommands(sb *strings.Builder, commands *Commands, indent int) {
	for _, commandname := range commands.nameindexes {
		command := commands.commandmap[commandname]
		writeIndent(sb, indent)
		sb.WriteString(commandname)
		if command.help != "" {
			sb.WriteRune('\t')
			sb.WriteString(command.help)
		}
		sb.WriteRune('\n')
		for _, paramlong := range command.Parameters.longindexes {
			param := command.Parameters.longparams[paramlong]
			shortparam := command.Parameters.longtoshort[paramlong]
			writeIndent(sb, indent)
			sb.WriteRune('\t')
			if param.required {
				if !param.raw {
					sb.WriteString("<--")
				} else {
					sb.WriteRune('<')
				}
				sb.WriteString(paramlong)
				sb.WriteRune('>')
			} else {
				if !param.raw {
					sb.WriteString("[--")
				} else {
					sb.WriteRune('[')
				}
				sb.WriteString(paramlong)
				sb.WriteRune(']')
			}
			if shortparam != "" {
				sb.WriteString("\t-")
				sb.WriteString(shortparam)
			}
			if param.value != nil {
				sb.WriteString("\t(")
				sb.WriteString(reflect.Indirect(reflect.ValueOf(param.value)).Type().Kind().String())
				sb.WriteRune(')')
			}
			if param.help != "" {
				sb.WriteRune('\t')
				sb.WriteString(param.help)
			}
			sb.WriteRune('\n')
		}
		sb.WriteRune('\n')
		if len(command.Commands.commandmap) > 0 {
			printCommands(sb, command.Commands, indent+1)
		}
	}
}

// writeIndent writes an indent string of n depth to sb.
func writeIndent(sb *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		sb.WriteRune('\t')
	}
}
