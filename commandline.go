// Copyright 2020 Vedran Vuk. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Package commandline implements a command line parser.
package commandline

import (
	"errors"
	"fmt"
	"reflect"
	"strings"

	"github.com/vedranvuk/strconvex"
)

// TODO Better printing. Use two spaces instead of tabs. Wrap to 80 chars.

var (
	// ErrNoArgs is returned by Parse if no arguments were specified on
	// command line and there are defined Commands or Params.
	ErrNoArgs = errors.New("commandline: no arguments")
	// ErrInvalidName is returned by Add* methods when an invalid Command,
	// Param long or Param short name is specified.
	ErrInvalidName = errors.New("commandline: invalid name")
	// ErrInvalidValue is returned by Add* and Parse methods if an invalid
	// parameter is given for a Param value, i.e. not a valid Go value pointer.
	ErrInvalidValue = errors.New("commandline: invalid value")
)

// Context is a CommandFunc context that provides info about Command execution.
type Context interface {
	// Name returns the name of Command which this handler was executed for.
	Name() string
	// Executed returns if this Command was the last matched Command and is
	// considered as executed. Matched Commands in the chain leading up to
	// last Command specified on command line are not marked as executed.
	//
	// All CommandFuncs defined for Commands in the command chain parsed from
	// command line arguments are visited and this property discerns the visited
	// Commands from the executed Command.
	Executed() bool
	// Parsed returns if the parameter under specified long name was parsed.
	// If the parameter under specified long name was not defined returns false.
	Parsed(string) bool
	// Arg returns the argument of parameter under specified long name or 
	// registered raw parameter.
	// If parameter was not parsed an empty string is returned.
	Arg(string) string
	// Args returns a slice of arguments passed to a Command in the order as
	// they were parsed if Command has no defined parameters and eturns an 
	// empty slice if Command has any defined parameters, prefixed or raw.
	//
	// It is used to retrieve arguments from a handler to implement custom
	// argument parsing.
	Args() []string
	// Print prints sub Commands of the Command this Context belongs to.
	Print() string
}

// CommandFunc is a prototype of a function that handles the event of a
// Command being parsed from command line arguments.
//
// CommandFuncs of all Commands in the chain parsed on command line are
// visited during Parser.Parse(). Only the last matched command is marked
// as executed and can be discerned from visited CommandFuncs using
// Context.Executed().
//
// If a CommandFunc returns a non-nil error further calling of parsed Commands
// CommandFuncs is aborted and the error is propagated to Parse method.
type CommandFunc = func(Context) error

// Parser is a command line parser. Its' Parse method is to be invoked
// with a slice of command line arguments passed to program.
// For example:
//  err := Parser.Parse(os.Args[1:])
//
// It is command oriented, meaning that one or more Command instances can be
// defined in Parser's Commands which when parsed from command line arguments
// invoke CommandFuncs registered for those Command instances. Command can have
// its' own Commands so a Command hierarchy can be defined.
//
// Root Commands, as an exception, allow for one Command with an empty name
// to be defined. This is to allow that program args need not start with a
// Command and to allow Params to be passed first which can act as "global".
// e.g. "--verbose list users".
//
// Command can have one or more Param instances defined in its' Params which
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
// Parser supports prefixed and raw params which can be combined on a Command
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
// Params require a Param Value. All combined short params must all be optional.
//
// Long Param names have the "--" prefix and cannot be combined.
//
// Raw params are not addressed but are instead matched against registered raw
// Params in order of registration as they appear on command line, respectively.
//
// Prefixed and raw params can both be registered for a Command but raw params
// must be registered last and specified after prefixed Params on the command
// line.
// e.g. "rmdir -v /home/me/stuff" where "rmdir" is a command, "-v" is a
// prefixed param and "/home/me/stuff" is a raw parameter.
//
// If Params are defined as optional they do not cause a parse error if not
// parsed from program args and return a parse error if defined as required and
// not parsed from command line.
//
// For specifics on how Params are parsed see AddParam and AddRawParam help.
//
// Commands can have a CommandFunc registered optionaly so that a Command can
// serve solely as sub-Command selector. For more details see CommandFunc.
//
// If no Params were defined on a Command all command line arguments following
// the command invocation will be passed to Command handler via Params.RawArgs.
//
// If no params were defined on a Command and the command has no CommandFunc
// registered an error is returned.
type Parser struct {
	// args is a slice of arguments being parsed.
	// Args are set once by Parse() then read and updated by Commands
	// and Params down the Parse chain until exhausted or an error occurs
	// using peek(), arg() and next().
	args []string
	// matchedCommands is a slice of commands parsed from command line in the
	// order as they were parsed.
	matchedCommands []*Command
	// Commands is the root command set.
	Commands
}

// New returns a new instance of *Parser.
func New() *Parser {
	p := &Parser{}
	p.Commands = *newCommands(p)
	return p
}

// Parse parses specified args, usually invoked as "Parse(os.Args[1:])".
// If a parse error occurs or an invoked Command handler returns an error
// it is returned. Returns ErrNoArgs if args are empty and there are defined
// Commands or Params.
func (p *Parser) Parse(args []string) error {
	p.args = args
	p.reset()
	return p.Commands.parse(p)
}

// Print prints the Parser as currently configured.
// Returns output suitable for terminal display.
func (p Parser) Print() string {
	sb := &strings.Builder{}
	printCommands(sb, &p.Commands, 0)
	return sb.String()
}

// writeIndent writes an indent string of n depth to sb.
func writeIndent(sb *strings.Builder, n int) {
	for i := 0; i < n; i++ {
		sb.WriteRune('\t')
	}
}

// printCommands is a recursive printer or registered Commands and Params.
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
		for _, paramlong := range command.Params.longindexes {
			param := command.Params.longparams[paramlong]
			shortparam := command.Params.longtoshort[paramlong]
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
			printCommands(sb, &command.Commands, indent+1)
		}
	}
}

// reset resets any Command and Param states prior to parsing.
func (p *Parser) reset() {
	p.matchedCommands = []*Command{}
	resetCommands(&p.Commands)
}

// resetCommands recursively resets all Commands and their Params states.
func resetCommands(c *Commands) {
	for _, cmd := range c.commandmap {
		cmd.executed = false
		cmd.Params.rawargs = []string{}
		if len(cmd.Params.longparams) > 0 {
			for _, p := range cmd.Params.longparams {
				p.parsed = false
				p.rawvalue = ""
			}
		}
		resetCommands(&cmd.Commands)
	}
}

// context is the Context adapter.
type context struct{ cmd *Command }

// Name returns Command's name.
func (c *context) Name() string { return c.cmd.name }

// Args returns raw Command arguments.
func (c *context) Args() []string { return c.cmd.Params.rawargs }

// Parsed returns if the long named parameter was parsed.
func (c *context) Parsed(name string) bool {
	if param, exists := c.cmd.Params.longparams[name]; exists {
		return param.parsed
	}
	return false
}

// Arg returns the argument of the raw parameter or prefixed parameter with
// specified long name.
func (c *context) Arg(name string) string {
	if param, exists := c.cmd.Params.longparams[name]; exists {
		return param.rawvalue
	}
	return ""
}

// Executed returns if the context's Command was executed or just visited.
func (c *context) Executed() bool { return c.cmd.executed }

// Print prints the context's Command.
func (c *context) Print() string { return c.cmd.print() }

// exec executes the context's command and returns its' handler return value.
func (c *context) exec() error {
	if c.cmd.f != nil {
		return c.cmd.f(c)
	}
	return nil
}

// visitCommands visits all matched commands, constructs a context and calls
// their handlers. Propagates first non-nil return value of visited handler.
func (p *Parser) visitCommands() error {
	var ctx context
	var l = len(p.matchedCommands)
	if l < 1 {
		return nil
	}
	var i int
	var err error
	for i = 0; i < l-1; i++ {
		ctx.cmd = p.matchedCommands[i]
		ctx.cmd.executed = false
		if err = ctx.exec(); err != nil {
			return err
		}
	}
	ctx.cmd = p.matchedCommands[i]
	ctx.cmd.executed = true
	return ctx.exec()
}

// argKind defines argument kind.
type argKind int

const (
	argInvalid      argKind = iota // Invalid argument.
	argNone                        // No argument.
	argCommandOrRaw                // Command or raw argument.
	argLong                        // Param with long name.
	argShort                       // Param with short name.
	argComb                        // Combined short Params.
)

// String implements stringer on argKind.
func (ak argKind) String() (s string) {
	switch ak {
	case argInvalid:
		s = "invalid"
	case argNone:
		s = "none"
	case argCommandOrRaw:
		s = "command or raw argument"
	case argLong:
		s = "long parameter"
	case argShort:
		s = "short parameter"
	case argComb:
		s = "combined short arguments"
	}
	return
}

// peek returns the first arg in args if args are not empty, otherwise returns
// an empty string.
func (p *Parser) peek() string {
	if len(p.args) > 0 {
		return p.args[0]
	}
	return ""
}

// next returns the first next in Parser trimmed of any prefixes and its' kind.
func (p *Parser) next() (arg string, kind argKind) {
	kind = argNone
	if len(p.args) == 0 {
		return
	}
	arg = p.args[0]
	if len(arg) == 0 {
		return
	}
	for i, l := 0, len(arg); i < l; i++ {
		if arg[0] != '-' {
			break
		}
		arg = arg[1:]
		if kind == argNone {
			kind = argShort
			continue
		}
		if kind == argShort {
			kind = argLong
			continue
		}
		if kind == argLong {
			return "", argInvalid
		}
	}
	if len(arg) == 0 {
		return "", argInvalid
	}
	if kind == argNone {
		kind = argCommandOrRaw
		return
	}
	if kind == argShort && len(arg) > 1 {
		return arg, argComb
	}
	return
}

// skip discards the first arg in the args slice and returns a bool indicating
// if there is any args left.
func (p *Parser) skip() bool {
	if len(p.args) == 0 {
		return false
	}
	p.args = p.args[1:]
	return len(p.args) > 0
}

// Command is a Command definition.
type Command struct {
	parent *Commands
	// help is the Command help text.
	help string
	// f is the function to invoke when this Command is executed.
	// Can be nil, CommandFunc or CommandRawFunc.
	f CommandFunc
	// executed specifies if this command was executed.
	executed bool
	Params   // Params are this Command's Params.
	Commands // Commands are this Command's Commands.
}

// newCommand returns a new *Command instance with given help and handler.
func newCommand(parent *Commands, help string, f CommandFunc) *Command {
	p := &Command{
		parent: parent,
		help:   help,
		f:      f,
	}
	p.Params = *newParams(p)
	p.Commands = *newCommands(p)
	return p
}

// print prints Commands contained in this Command.
func (c *Command) print() string {
	sb := &strings.Builder{}
	printCommands(sb, &c.Commands, 0)
	return sb.String()
}

// nameToCommand is a map of command name to *Command.
type nameToCommand map[string]*Command

// Commands holds a set of Commands with a unique name.
type Commands struct {
	// parent is this Commands parent.
	// If it is a Parser this Commands is the root Commands.
	// If it is a Command this is a sub-Command Commands.
	parent interface{}
	// name is the Command name.
	name string
	// commandmap is a map of command names to *Command definitions.
	commandmap nameToCommand
	// nameindexes is a slice of command names in order as they were defined.
	nameindexes []string
}

// newCommands returns a new *Commands instance owned by parent.
func newCommands(parent interface{}) *Commands {
	return &Commands{
		parent:     parent,
		commandmap: make(nameToCommand),
	}
}

// AddCommand registers a new Command under specified name and help text that
// invokes f when parsed from arguments if it is not nil. Name is the only
// required parameter.
//
// If Commands is the root set in a Parser it can register a single Command
// with an empty name to be an unnamed container in a global params pattern.
//
// If a registration error occurs it is returned with a nil *Command.
func (c *Commands) AddCommand(name, help string, f CommandFunc) (*Command, error) {
	// Allow empty Command name only in root.
	if name == "" {
		if _, ok := c.parent.(*Parser); !ok {
			return nil, ErrInvalidName
		}
	}
	// No duplicate names.
	if _, exists := c.commandmap[name]; exists {
		if name == "" {
			return nil, errors.New("commandline: duplicate empty root command")
		}
		return nil, fmt.Errorf("commandline: duplicate command name '%s'", name)
	}
	// Disallow adding sub-Commands to a Command with raw args.
	if parentcmd, ok := c.parent.(*Command); ok {
		if parentcmd.Params.hasRawArgs() {
			return nil, errors.New("commandline: cannot register a sub-command in a command with raw parameters")
		}
	}
	// Define and add a new Command to self.
	cmd := newCommand(c, help, f)
	cmd.name = name
	c.commandmap[name] = cmd
	c.nameindexes = append(c.nameindexes, name)
	return cmd, nil
}

// MustAddCommand is like AddCommand except the function panics on error.
// Returns added *Command.
func (c *Commands) MustAddCommand(name, help string, f CommandFunc) *Command {
	cmd, err := c.AddCommand(name, help, f)
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
	if cmd, ok := c.commandmap[name]; ok {
		return cmd
	}
	panic(fmt.Sprintf("commandline: command '%s' not found", name))
}

// parse parses Parser args into this Commands.
func (c *Commands) parse(cl *Parser) error {
	var cmd *Command
	var exists, global bool
	switch arg, kind := cl.next(); kind {
	case argInvalid:
		return errors.New("commandline: invalid argument")
	case argNone:
		// Execute last matched Command.
		if len(cl.matchedCommands) > 0 {
			return cl.visitCommands()
		}
		// Otherwise, nothing was matched so far.
		return ErrNoArgs
	case argCommandOrRaw:
		// Arg should otherwise be an existing Command.
		if cmd, exists = c.commandmap[arg]; !exists {
			// See if last matched command has no defined params
			// and execute it with stored raw params.

			if len(cl.matchedCommands) > 0 {
				if cmd = cl.matchedCommands[len(cl.matchedCommands)-1]; cmd.f != nil && cmd.paramCount() == 0 {
					return cl.visitCommands()
				}
			}
			// Else, this is extra.
			return fmt.Errorf("commandline: command '%s' not found", arg)
		}
	default:
		// Arg is not a Command, but a Param. See if a
		// special case of single unnamed root command.
		cmd, exists = c.commandmap[""]
		if !exists {
			return fmt.Errorf("commandline: expected command, got %s '%s'", kind, arg)
		}
		global = true
	}
	// Advance to next arg, stop if no more.
	if !global {
		cl.skip()
	}
	// Parse Params.
	if err := cmd.Params.parse(cl); err != nil {
		return err
	}
	// Append command to matched commands.
	cl.matchedCommands = append(cl.matchedCommands, cmd)
	// Repeat parse on these Commands if "global params"
	// empty Command name container was invoken.
	if global {
		return c.parse(cl)
	}
	// Or pass control to contained Commands.
	return cmd.Commands.parse(cl)
}

// parser returns the parser these Commands belong to.
func (c *Commands) parser() (p *Parser) {
	var cmd *Command
	var ok bool
	if cmd, ok = c.parent.(*Command); ok {
		return cmd.parent.parser()
	}
	if p, ok = c.parent.(*Parser); ok {
		return p
	}
	panic("commands have no parent parser")
}

// Param defines a Command parameter contained in a Params.
type Param struct {
	// help is the Param help text.
	help string
	// required specifies if this Param is required.
	required bool
	// rawvalue is the raw parsed param value, possibly empty.
	rawvalue string
	// value is a pointer to a Go value which is set
	// from parsed Param value if not nil and points to a
	// valid target.
	value interface{}
	// raw specifies if this param is a raw param.
	raw bool
	// parsed indicates if Param was parsed from arguments.
	parsed bool
}

// newParam returns a new *Param instance with given help, required and value.
func newParam(help string, required, raw bool, value interface{}) *Param {
	return &Param{
		help:     help,
		required: required,
		raw:      raw,
		value:    value,
	}
}

// nameToParam maps a param name to *Param.
type nameToParam map[string]*Param

// longToShort maps a long param name to short param name.
type longToShort map[string]string

// A Params defines a set of Command Params unique by long name.
type Params struct {
	// cmd is the reference to owner *Command.
	cmd *Command
	// longparams is a map of long param name to *Param.
	longparams nameToParam
	// shortparams is a map of short param name to *Param.
	shortparams nameToParam
	// longtoshort maps a long param name to short param name.
	longtoshort longToShort
	// longindexes hold long param names in order as they are added.
	longindexes []string
	// rawargs stores the parsed raw Param instances.
	rawargs []string
}

// newParams returns a new instance of *Params.
func newParams(cmd *Command) *Params {
	return &Params{
		cmd,
		make(nameToParam),
		make(nameToParam),
		make(longToShort),
		[]string{},
		[]string{},
	}
}

// AddParam registers a new Param in these Params.
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
// If an error occurs Param is not registered.
func (p *Params) AddParam(long, short, help string, required bool, value interface{}) error {
	return p.addParam(long, short, help, required, false, value)
}

// MustAddParam is like AddParam except the function panics on error.
// Returns a Command that the param was added to.
func (p *Params) MustAddParam(long, short, help string, required bool, value interface{}) *Command {
	if err := p.AddParam(long, short, help, required, value); err != nil {
		panic(err)
	}
	return p.cmd
}

// AddRawParam registers a raw Param under specified name which must be unique
// in long Params names. Raw params can only be defined after prefixed params
// or other raw params. Calls to AddParam after AddRawParam will error.
//
// Parsed arguments are applied to registered raw Params in order as they are
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
func (p *Params) AddRawParam(name, help string, required bool, value interface{}) error {
	return p.addParam(name, "", help, required, true, value)
}

// MustAddRawParam is like AddRawParam except the function panics on error.
// Returns a Command that the param was added to.
func (p *Params) MustAddRawParam(name, help string, required bool, value interface{}) *Command {
	if err := p.AddRawParam(name, help, required, value); err != nil {
		panic(err)
	}
	return p.cmd
}

// hasRawArgs returns if Params contain one or more raw Param instances.
func (p *Params) hasRawArgs() bool {
	for _, par := range p.longparams {
		if par.raw {
			return true
		}
	}
	return false
}

// addParam is the implementation of AddParam minus the check of adding a
// normal Param to a Command with a CommandRawFunc handler.
func (p *Params) addParam(long, short, help string, required, raw bool, value interface{}) error {
	// Long name must not be empty and short name must be max one char long.
	if long == "" || len(short) > 1 {
		return ErrInvalidName
	}
	// No long duplicates.
	if _, exists := p.longparams[long]; exists {
		return fmt.Errorf("commandline: duplicate long parameter name '%s'", long)
	}
	// No short duplicates if not empty.
	if _, exists := p.shortparams[short]; exists && short != "" {
		return fmt.Errorf("commandline: duplicate short parameter name '%s'", short)
	}
	// Raw params can only be registered after prefixed params.
	// Optional raw params can only be registered after required raw params.
	if lp := p.last(); lp != nil {
		if lp.raw {
			if !raw {
				return errors.New("commandline: cannot register prefixed parameter after raw parameter")
			}
			if !lp.required && !required {
				return errors.New("commandline: cannot register multiple optional parameters")
			}
			if !lp.required && required {
				return errors.New("commandline: cannot register required after optional parameter")
			}
		}
	}
	// Required prefixed params need a valid Go value.
	if value == nil {
		if !raw && required {
			return errors.New("commandline: value required")
		}
	} else {
		// Value must be a valid pointer to a Go value.
		if v := reflect.ValueOf(value); !v.IsValid() || v.Kind() != reflect.Ptr {
			return ErrInvalidValue
		}
	}
	// Register a new param.
	param := newParam(help, required, raw, value)
	p.longparams[long] = param
	if short != "" {
		p.shortparams[short] = param
	}
	p.longtoshort[long] = short
	p.longindexes = append(p.longindexes, long)
	return nil
}

// last returns the last defined arg or nil if none registered.
func (p *Params) last() *Param {
	if len(p.longindexes) == 0 {
		return nil
	}
	return p.longparams[p.longindexes[len(p.longindexes)-1]]
}

// paramCount returns number of defined params.
func (p *Params) paramCount() int { return len(p.longindexes) }

// parse parses the Parser args into this Params.
func (p *Params) parse(cl *Parser) error {
	var err error
	var arg string
	var kind argKind
	var count int = p.paramCount()
	var param *Param
	var exists bool
	// No defined params.
	if count == 0 {
		// Last Command in chain.
		if len(p.cmd.commandmap) == 0 {
			// If there are any args left store them as raw arguments.
			if len(cl.args) > 0 {
				p.rawargs = append(p.rawargs, cl.args...)
			}
			// Last command in chain must have a handler.
			if p.cmd.f == nil {
				return errors.New("commandline: no handler for arguments")
			}
		}
		return nil
	}
	for i := 0; i < count; {
		arg, kind = cl.next()
		switch kind {
		case argInvalid:
			return errors.New("commandline: invalid argument")
		case argNone:
			// Nothing to parse.
			goto check
		case argCommandOrRaw:
			// Only raw params possibly accept non-prefixed arguments.
			// Check if there are any named params left, advance past
			// them then try parsing the arg into first raw param.
			// Commands with raw args cannot have sub commands.
			for i < count {
				if param = p.longparams[p.longindexes[i]]; !param.raw {
					i++
					continue
				}
				break
			}
			// If there are no (more) raw params defined,
			// return to Command parsing.
			if i >= count {
				return nil
			}
			i++
		case argShort:
			if param, exists = p.shortparams[arg]; !exists {
				return fmt.Errorf("commandline: short parameter '%s' not found", arg)
			}
			i++
		case argLong:
			if param, exists = p.longparams[arg]; !exists {
				return fmt.Errorf("commandline: long parameter '%s' not found", arg)
			}
			i++
		case argComb:
			// Parse all combined args and return.
			shorts := strings.Split(arg, "")
			for _, short := range shorts {
				if param, exists = p.shortparams[short]; !exists {
					return fmt.Errorf("commandline: short parameter '%s' not found", short)
				}
				if param.value != nil {
					return fmt.Errorf("commandline: short parameter '%s' requires argument, cannot combine", short)
				}
				// Param is specified multiple times.
				if param.parsed {
					return fmt.Errorf("commandline: parameter '%s' specified multiple times", short)
				}
				param.parsed = true
				i++
			}
			cl.skip()
			continue
		}
		// Param is specified multiple times.
		if param.parsed {
			return fmt.Errorf("commandline: parameter '%s' specified multiple times", arg)
		}
		// Parse value argument for params with value.
		if param.value != nil {
			// Advance argument for prefixed params.
			if !param.raw {
				if !cl.skip() {
					return fmt.Errorf("commandline: parameter '%s' requires a value", arg)
				}
				arg = cl.peek()
			}
			// Set value.
			if err = stringToGoValue(arg, param.value); err != nil {
				return err
			}
		}
		// Advance.
		param.rawvalue = arg
		param.parsed = true
		if !cl.skip() {
			break
		}
	}
check:
	// Check all required params were parsed.
	for arg, param = range p.longparams {
		if param.required && !param.parsed {
			return fmt.Errorf("commandline: required parameter '%s' not specified", arg)
		}
	}
	return nil
}

// stringToGoValue converts a string to a Go value or returns an error.
func stringToGoValue(s string, i interface{}) error {
	if err := strconvex.StringToInterface(s, i); err != nil {
		return fmt.Errorf("commandline: error converting value %s: %w", s, err)
	}
	return nil
}
